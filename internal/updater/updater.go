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
	"cloud.google.com/go/translate"
	"cloud.google.com/go/vision/v2/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"

	"firebase.google.com/go"

	"golang.org/x/text/language"
)

const (
	TopicID             = "update-wallpapers-v2"
	SubID               = "update-wallpapers-v2-sub"
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
		"en-IN",
	}

	nonENMarkets = []string{
		"fr-FR",
		"de-DE",
		"es-ES",
		"zh-CN",
		"ja-JP",
		"it-IT",
		"pt-BR",
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
		firestoreClient: firestoreClient,
		httpClient:      httpClient,
		imageClient:     imageClient,
		translateClient: translateClient,
	}, nil
}

type Updater struct {
	annoClient      *vision.ImageAnnotatorClient
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

		// if exists, check if we need to update
		if existingImage != nil {
			// Check if we need to update: either missing urlBase OR upgrading to English
			needsURLBaseUpdate := existingImage.URLBase == ""
			needsMarketUpgrade := slices.Contains(nonENMarkets, existingImage.Market) && slices.Contains(ENMarkets, image.Market)

			if !needsURLBaseUpdate && !needsMarketUpgrade {
				continue
			}

			// maintain old date
			image.Date = existingImage.Date

			// If only updating urlBase (not upgrading market), preserve existing fields
			if needsURLBaseUpdate && !needsMarketUpgrade {
				existingImage.URLBase = image.URLBase
				_, err = u.firestoreClient.Upsert(ctx, *existingImage)
			} else {
				// Full update (market upgrade case)
				_, err = u.firestoreClient.Upsert(ctx, image)
			}

			if err != nil {
				return err
			}
			updatedImages = append(updatedImages, image.ID)

			continue
		}

		fmt.Printf("%s new wallpaper found\n", image.ID)

		// translate title if not english
		if slices.Contains(nonENMarkets, image.Market) {
			translatedTitle, err := u.translateText(ctx, image.Title)
			if err != nil {
				return err
			} else if translatedTitle != "" {
				image.Title = translatedTitle
			}
		}

		anno, err := u.annotateImage(ctx, image.URL(bingURL))
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
	// Download image first since Vision API can't access Bing URLs directly
	imgReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create image request: %w", err)
	}

	imgResp, err := u.httpClient.Do(imgReq)
	if err != nil {
		return nil, fmt.Errorf("failed to download image from %s: %w", url, err)
	}
	defer imgResp.Body.Close()

	if imgResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image from %s: status %d", url, imgResp.StatusCode)
	}

	imgBytes, err := io.ReadAll(imgResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image from %s: %w", url, err)
	}

	req := &visionpb.BatchAnnotateImagesRequest{
		Requests: []*visionpb.AnnotateImageRequest{
			{
				Image: &visionpb.Image{
					Content: imgBytes,
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

func (u *Updater) fetchAndDedupeImages(ctx context.Context, markets []string, out map[string]imgpkg.Image) error {
	for _, market := range markets {
		bi, err := u.imageClient.List(ctx, market)
		if err != nil {
			return err
		}

		for _, v := range bi {
			if !v.WP {
				continue
			}

			image, err := imgpkg.From(v, market)
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
