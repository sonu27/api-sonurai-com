package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"api/internal/updater/bing_image"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"cloud.google.com/go/translate"
	"cloud.google.com/go/vision/apiv1"
	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	"firebase.google.com/go"
	"golang.org/x/exp/slices"
	"golang.org/x/text/language"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	firestoreClient, err := firebaseClient.Firestore(ctx)
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
	images := make(map[string]Image)

	if err := u.fetchAndDeduplicateImages(ctx, ENMarkets, images); err != nil {
		return err
	}
	if err := u.fetchAndDeduplicateImages(ctx, nonENMarkets, images); err != nil {
		return err
	}

	// for each wallpaper, check if exists in db
	fmt.Printf("%d images found\n", len(images))
	for _, v := range images {
		if !u.fileExists(v.URL) {
			continue
		}

		dsnap, err := u.firestoreClient.Collection(firestoreCollection).Doc(v.ID).Get(ctx)
		if status.Code(err) == codes.NotFound {
			fmt.Printf("%s new wallpaper found\n", v.ID)
		}

		if dsnap.Exists() {
			b, err := json.Marshal(dsnap.Data())
			if err != nil {
				return err
			}

			var result Image
			if err := json.Unmarshal(b, &result); err != nil {
				return err
			}

			if slices.Contains(nonENMarkets, result.Market) && slices.Contains(ENMarkets, v.Market) {
				// maintain old date
				v.Date = result.Date
			} else {
				continue
			}
		}

		// todo: add retry if error
		if err := u.downloadFile(ctx, v.URL, v.Filename+".jpg"); err != nil {
			return err
		}

		if slices.Contains(nonENMarkets, v.Market) {
			translatedTitle, err := u.translateText(ctx, v.Title)
			if err != nil {
				return err
			} else if translatedTitle != "" {
				v.Title = translatedTitle
			}
		}

		var wallpaper map[string]any
		inrec, _ := json.Marshal(v)
		err = json.Unmarshal(inrec, &wallpaper)
		if err != nil {
			return err
		}

		_, err = u.updateWallpaper(ctx, v.ID, wallpaper)
		if err != nil {
			return err
		}
		updatedImages = append(updatedImages, v.ID)

		// add extras info e.g. labels
		if !dsnap.Exists() {
			anno, err := u.detectLabels(ctx, v.ThumbURL)
			if err != nil {
				fmt.Println(err.Error())
			}

			tags := make(map[string]float32)
			for _, v := range anno {
				tags[strings.ToLower(v.Description)] = v.Score
			}

			gg := map[string]any{
				"tags": tags,
			}

			_, err = u.updateWallpaper(ctx, v.ID, gg)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}

	fmt.Printf("%d images updated: %s\n", len(updatedImages), strings.Join(updatedImages, ", "))
	return nil
}

func (u *Updater) detectLabels(ctx context.Context, url string) ([]*visionpb.EntityAnnotation, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	image, err := vision.NewImageFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	annotations, err := u.annoClient.DetectLabels(ctx, image, nil, 50)
	if err != nil {
		return nil, err
	}

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
	objWriter.Close()
	return nil
}

func (u *Updater) fetchAndDeduplicateImages(ctx context.Context, markets []string, out map[string]Image) error {
	for _, market := range markets {
		bi, err := u.imageClient.List(ctx, market)
		if err != nil {
			return err
		}

		for _, v := range bi {
			image, err := convertToImage(v, market)
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

func (u *Updater) updateWallpaper(ctx context.Context, ID string, data map[string]any) (*firestore.WriteResult, error) {
	return u.firestoreClient.Collection(firestoreCollection).Doc(ID).Set(ctx, data, firestore.MergeAll)
}

func convertToImage(bw bing_image.Image, market string) (Image, error) {
	fullDesc := bw.Copyright
	id := strings.Replace(bw.URLBase, "/az/hprichbg/rb/", "", 1)
	filename := strings.Replace(id, "/th?id=OHR.", "", 1)
	id = strings.Split(filename, "_")[0]

	date, err := strconv.Atoi(bw.StartDate)
	if err != nil {
		return Image{}, err
	}

	var copyright string
	var title string

	if a := strings.Split(bw.Copyright, "（©"); len(a) == 2 {
		// chinese chars
		title = a[0]
		copyright = "© " + a[1]
		copyright = strings.Replace(copyright, "）", "", 1)
	} else if a := strings.Split(bw.Copyright, "(©"); len(a) == 2 {
		title = a[0]
		copyright = "© " + a[1]
		copyright = strings.Replace(copyright, ")", "", 1)
	} else {
		a := strings.Split(bw.Copyright, "©")
		title = a[0]
		copyright = "© " + a[1]
		copyright = strings.Replace(copyright, ")", "", 1)
	}

	title = strings.TrimSpace(title)
	copyright = strings.TrimSpace(copyright)

	image := Image{
		ID:        id,
		Title:     title,
		Copyright: copyright,
		Date:      date,
		Filename:  filename,
		Market:    market,
		FullDesc:  fullDesc,
		URL:       bingURL + bw.URLBase + "_1920x1200.jpg",
		ThumbURL:  bingURL + bw.URLBase + "_1920x1080.jpg",
	}

	return image, nil
}
