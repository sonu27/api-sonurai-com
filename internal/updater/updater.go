package updater

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"api/internal/updater/bing"
	"api/internal/updater/firestore"
	imgpkg "api/internal/updater/image"
	"cloud.google.com/go/storage"
	"cloud.google.com/go/translate"
	"cloud.google.com/go/vision/v2/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"

	"firebase.google.com/go"

	"golang.org/x/text/language"
)

const (
	TopicID             = "update-wallpapers-v2"
	SubID               = "update-wallpapers-v2-sub"
	bucketName          = "images.sonurai.com"
	firestoreCollection = "BingWallpapers"
	bingURL             = "https://www.bing.com"
)

var (
	ENMarkets = []string{
		"en-GB",
		"en-US",
		"en-CA",
		"en-AU",
		"en-NZ",
	}

	nonENMarkets = []string{
		"fr-FR",
		"de-DE",
		"es-ES",
		"zh-CN",
		"ja-JP",
	}
)

func New() (*Updater, error) {
	ctx := context.Background()
	conf := &firebase.Config{ProjectID: os.Getenv("PROJECT_ID")}

	httpClient := &http.Client{Timeout: time.Second * 15}

	imageClient := &bing.Client{BaseURL: bingURL, HC: httpClient}

	firebaseClient, err := firebase.NewApp(ctx, conf)
	if err != nil {
		return nil, err
	}

	firestoreClient, err := firestore.NewClient(ctx, firestoreCollection, firebaseClient)
	if err != nil {
		return nil, err
	}

	storageClient, err := firebaseClient.Storage(ctx)
	if err != nil {
		return nil, err
	}

	bucket, err := storageClient.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	if _, err := bucket.Attrs(ctx); err != nil {
		return nil, err
	}

	annoClient, err := vision.NewImageAnnotatorClient(ctx)
	if err != nil {
		return nil, err
	}

	translateClient, err := translate.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &Updater{
		annoClient:      annoClient,
		bucket:          bucket,
		firestoreClient: firestoreClient,
		httpClient:      httpClient,
		imageClient:     imageClient,
		translateClient: translateClient,
	}, nil
}

type Updater struct {
	annoClient      *vision.ImageAnnotatorClient
	bucket          *storage.BucketHandle
	firestoreClient *firestore.Client
	httpClient      *http.Client
	imageClient     *bing.Client
	translateClient *translate.Client
}

func (u *Updater) Update(ctx context.Context) error {
	fmt.Println("updating images")

	var updatedImages []string
	images := make(map[string]imgpkg.Image)

	if err := u.fetchAndDedupeImages(ctx, append(ENMarkets, nonENMarkets...), images); err != nil {
		return err
	}

	// for each wallpaper, check if exists in db
	fmt.Printf("%d images found\n", len(images))
	for _, image := range images {
		existingImage, err := u.firestoreClient.Get(ctx, image.ID)
		if err != nil {
			return err
		}

		// if exists as non-english, and new existingImage is english, update it
		if existingImage != nil {
			if !slices.Contains(nonENMarkets, existingImage.Market) || !slices.Contains(ENMarkets, image.Market) {
				continue
			}

			// maintain old date
			image.Date = existingImage.Date

			_, err = u.firestoreClient.Upsert(ctx, image)
			if err != nil {
				return err
			}
			updatedImages = append(updatedImages, image.ID)

			continue
		}

		if !u.fileExists(image.URL) {
			continue
		}

		fmt.Printf("%s new wallpaper found\n", image.ID)

		// todo: add retry if error
		if err := u.downloadFile(ctx, image.URL, image.Filename+".jpg"); err != nil {
			return err
		}

		// translate title if not english
		if slices.Contains(nonENMarkets, image.Market) {
			translatedTitle, err := u.translateText(ctx, image.Title)
			if err != nil {
				return err
			} else if translatedTitle != "" {
				image.Title = translatedTitle
			}
		}

		anno, err := u.annotateImage(ctx, image.Filename+".jpg")
		if err != nil {
			return err
		}

		// Process label annotations
		for _, v := range anno.GetLabelAnnotations() {
			image.Tags[strings.ToLower(v.Description)] = v.Score
		}

		// start: duplicate tags to t
		tmp := make([]tag, 0, len(image.Tags))
		for k, v := range image.Tags {
			tmp = append(tmp, tag{Name: k, Score: v})
		}

		// sort by score
		sort.SliceStable(tmp, func(i, j int) bool {
			return tmp[i].Score > tmp[j].Score
		})

		image.TagsOrdered = make([]string, 0, len(tmp))
		for _, v := range tmp {
			image.TagsOrdered = append(image.TagsOrdered, strings.ReplaceAll(v.Name, " ", "-"))
		}
		// end: duplicate tags to t

		// Extract up to 4 dominant colors as hex strings
		if props := anno.GetImagePropertiesAnnotation(); props != nil {
			if dc := props.GetDominantColors(); dc != nil {
				colors := dc.GetColors()
				for i := range min(4, len(colors)) {
					c := colors[i].GetColor()
					rgb := imgpkg.RGB{int(c.GetRed()), int(c.GetGreen()), int(c.GetBlue())}
					image.Colors = append(image.Colors, rgb.ToHex())
				}
			}
		}

		_, err = u.firestoreClient.Upsert(ctx, image)
		if err != nil {
			return err
		}
		updatedImages = append(updatedImages, image.ID)
	}

	fmt.Printf("%d images updated: %s\n", len(updatedImages), strings.Join(updatedImages, ", "))
	return nil
}

func (u *Updater) annotateImage(ctx context.Context, url string) (*visionpb.AnnotateImageResponse, error) {
	url = fmt.Sprintf("gs://%s/%s", bucketName, url)
	req := &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Source: &visionpb.ImageSource{
						GcsImageUri: url,
					},
				},
				Features: []*visionpb.Feature{
					{
						Type:       visionpb.Feature_LABEL_DETECTION,
						MaxResults: 50,
					},
					{
						Type:       visionpb.Feature_IMAGE_PROPERTIES,
						MaxResults: 4,
					},
				},
			},
		},
	}
	images, err := u.annoClient.BatchAnnotateImages(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vision API call failed: %w", err)
	}

	responses := images.GetResponses()
	if len(responses) == 0 {
		return nil, fmt.Errorf("vision API returned no responses for %s", url)
	}

	resp := responses[0]
	if resp.GetError() != nil {
		return nil, fmt.Errorf("vision API error for %s: %s", url, resp.GetError().GetMessage())
	}

	return resp, nil
}

func (u *Updater) downloadFile(ctx context.Context, url string, name string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: unexpected status %d", resp.StatusCode)
	}

	objWriter := u.bucket.Object(name).NewWriter(ctx)

	_, err = io.Copy(objWriter, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if err = objWriter.Close(); err != nil {
		return fmt.Errorf("failed to close object writer: %w", err)
	}

	return nil
}

func (u *Updater) fetchAndDedupeImages(ctx context.Context, markets []string, out map[string]imgpkg.Image) error {
	for _, market := range markets {
		bi, err := u.imageClient.List(ctx, market)
		if err != nil {
			return err
		}

		for _, v := range bi {
			image, err := imgpkg.From(v, market, bingURL)
			if err != nil {
				return err
			}

			if _, ok := out[image.ID]; !ok {
				out[image.ID] = image
			}
		}
	}
	return nil
}

func (u *Updater) fileExists(url string) bool {
	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return false
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func (u *Updater) translateText(ctx context.Context, text string) (string, error) {
	lang, _ := language.Parse("en")
	opts := &translate.Options{Format: "text"}
	resp, err := u.translateClient.Translate(ctx, []string{text}, lang, opts)
	if err != nil {
		return "", fmt.Errorf("translate: %v", err)
	}
	if len(resp) == 0 {
		return "", fmt.Errorf("translate returned empty response to text: %s", text)
	}

	return resp[0].Text, nil
}

type tag struct {
	Name  string
	Score float32
}
