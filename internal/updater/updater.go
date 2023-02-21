package updater

import (
	"api/internal/updater/bing_image"
	"api/internal/updater/firestore"
	"api/internal/updater/image"
	"cloud.google.com/go/storage"
	"cloud.google.com/go/translate"
	"cloud.google.com/go/vision/v2/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	"context"
	"firebase.google.com/go"
	"fmt"
	"golang.org/x/exp/slices"
	"golang.org/x/text/language"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
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

	imageClient := &bing_image.Client{BaseURL: bingURL, HC: httpClient}

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
	imageClient     *bing_image.Client
	translateClient *translate.Client
}

func (u *Updater) Update(ctx context.Context) error {
	fmt.Println("updating images")

	var updatedImages []string
	images := make(map[string]image.Image)

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

		anno, err := u.detectLabels(ctx, image.Filename+".jpg")
		if err != nil {
			return err
		}

		for _, v := range anno {
			image.Tags[strings.ToLower(v.Description)] = v.Score
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

func (u *Updater) detectLabels(ctx context.Context, url string) ([]*visionpb.EntityAnnotation, error) {
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
				},
			},
		},
	}
	images, err := u.annoClient.BatchAnnotateImages(ctx, req)
	if err != nil {
		return nil, err
	}

	annotations := images.GetResponses()[0].GetLabelAnnotations()

	return annotations, nil
}

func (u *Updater) downloadFile(ctx context.Context, url string, name string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	objWriter := u.bucket.Object(name).NewWriter(ctx)

	_, err = io.Copy(objWriter, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	err = objWriter.Close()
	if err != nil {
		return err
	}

	return nil
}

func (u *Updater) fetchAndDedupeImages(ctx context.Context, markets []string, out map[string]image.Image) error {
	for _, market := range markets {
		bi, err := u.imageClient.List(ctx, market)
		if err != nil {
			return err
		}

		for _, v := range bi {
			image, err := image.From(v, market, bingURL)
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
	req, _ := http.NewRequest(http.MethodHead, url, nil)
	resp, _ := u.httpClient.Do(req)

	return resp.StatusCode == 200
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
