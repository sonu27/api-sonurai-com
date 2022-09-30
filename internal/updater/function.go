package updater

import (
	"context"
	"encoding/base64"
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
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"cloud.google.com/go/translate"
	vision "cloud.google.com/go/vision/apiv1"
	firebase "firebase.google.com/go"
	"github.com/mitchellh/mapstructure"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
	vision2 "google.golang.org/genproto/googleapis/cloud/vision/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	projectID           = "sonurai-com-v3"
	topicID             = "update-wallpapers-v2"
	bucketName          = "images.sonurai.com"
	firestoreCollection = "BingWallpapers"
	bingURL             = "https://www.bing.com"
)

var (
	httpClient      *http.Client
	imageClient     *bing_image.Client
	annoClient      *vision.ImageAnnotatorClient
	firestoreClient *firestore.Client
	translateClient *translate.Client

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

type Image struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Copyright string `json:"copyright"`
	Date      int    `json:"date"`
	Filename  string `json:"filename"`
	Market    string `json:"market"`
	FullDesc  string `json:"fullDesc"`
	URL       string `json:"url"`
	ThumbURL  string `json:"thumbUrl"`
}

type PubSubMessage struct {
	Data []byte `json:"data"`
}

func New(test bool, firebase *firebase.App, firestore *firestore.Client) error {
	ctx := context.Background()

	httpClient = &http.Client{Timeout: time.Second * 5}
	imageClient = &bing_image.Client{HC: httpClient}

	firestoreClient = firestore

	storageClient, err := firebase.Storage(ctx)
	if err != nil {
		return err
	}

	bucket, err := storageClient.Bucket(bucketName)
	if err != nil {
		return err
	}

	if _, err = bucket.Attrs(ctx); err != nil {
		return err
	}

	saJSON, err := base64.StdEncoding.DecodeString(os.Getenv("FIRESTORE_SA"))
	if err != nil {
		return err
	}
	sa := option.WithCredentialsJSON(saJSON)

	annoClient, err = vision.NewImageAnnotatorClient(ctx, sa)
	if err != nil {
		return err
	}

	translateClient, err = translate.NewClient(ctx, sa)
	if err != nil {
		return err
	}

	if test {
		return Start(ctx, bucket)
	}

	pubsubClient, err := pubsub.NewClient(ctx, projectID, sa)
	if err != nil {
		return err
	}
	defer pubsubClient.Close()

	topic, err := getOrCreateTopic(ctx, pubsubClient, topicID)
	if err != nil {
		return err
	}

	sub, err := getOrCreateSub(ctx, pubsubClient, "sub1", &pubsub.SubscriptionConfig{
		Topic:                     topic,
		EnableExactlyOnceDelivery: true,
	})
	if err != nil {
		return err
	}

	return sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		msg.Ack()
		err := Start(ctx, bucket)
		if err != nil {
			fmt.Println(err)
		}
	})
}

func Start(ctx context.Context, bucket *storage.BucketHandle) error {
	var updatedImages []string
	images := make(map[string]Image)

	if err := fetchAndDeduplicateImages(ctx, ENMarkets, images); err != nil {
		return err
	}
	if err := fetchAndDeduplicateImages(ctx, nonENMarkets, images); err != nil {
		return err
	}

	// for each wallpaper, check if exists in db
	fmt.Printf("%d images found\n", len(images))
	for _, v := range images {
		if !fileExists(v.URL) {
			continue
		}

		dsnap, err := firestoreClient.Collection(firestoreCollection).Doc(v.ID).Get(ctx)
		if status.Code(err) == codes.NotFound {
			fmt.Printf("%s new wallpaper found\n", v.ID)
		}

		if dsnap.Exists() {
			var result Image
			mapstructure.Decode(dsnap.Data(), &result)

			if stringInSlice(result.Market, nonENMarkets) && stringInSlice(v.Market, ENMarkets) {
				// maintain old date
				v.Date = result.Date
			} else {
				continue
			}
		}

		downloadFile(ctx, bucket, v.URL, v.Filename+".jpg")

		if stringInSlice(v.Market, nonENMarkets) {
			translatedTitle, err := translateText(ctx, v.Title)
			if err != nil {
				return err
			} else if translatedTitle != "" {
				v.Title = translatedTitle
			}
		}

		var wallpaper map[string]interface{}
		inrec, _ := json.Marshal(v)
		err = json.Unmarshal(inrec, &wallpaper)
		if err != nil {
			return err
		}

		_, err = updateWallpaper(ctx, v.ID, wallpaper)
		if err != nil {
			return err
		}
		updatedImages = append(updatedImages, v.ID)

		// add extras info e.g. labels
		if !dsnap.Exists() {
			anno, err := detectLabels(ctx, v.ThumbURL)
			if err != nil {
				fmt.Println(err.Error())
			}

			tags := make(map[string]float32)
			for _, v := range anno {
				tags[strings.ToLower(v.Description)] = v.Score
			}

			gg := map[string]interface{}{
				"tags": tags,
			}

			_, err = updateWallpaper(ctx, v.ID, gg)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}

	fmt.Printf("%d updated images %s\n", len(updatedImages), strings.Join(updatedImages, ", "))

	return nil
}

func fetchAndDeduplicateImages(ctx context.Context, markets []string, out map[string]Image) error {
	for _, market := range markets {
		bi, err := imageClient.List(ctx, market)
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

func translateText(ctx context.Context, text string) (string, error) {
	lang, _ := language.Parse("en")
	opts := &translate.Options{
		Format: "text",
	}
	resp, err := translateClient.Translate(ctx, []string{text}, lang, opts)
	if err != nil {
		return "", fmt.Errorf("translate: %v", err)
	}
	if len(resp) == 0 {
		return "", fmt.Errorf("translate returned empty response to text: %s", text)
	}

	return resp[0].Text, nil
}

func fileExists(url string) bool {
	req, _ := http.NewRequest(http.MethodHead, url, nil)
	resp, _ := httpClient.Do(req)

	return resp.StatusCode == 200
}

func downloadFile(ctx context.Context, bucket *storage.BucketHandle, url string, name string) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fmt.Println(err.Error())
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer resp.Body.Close()
	objWriter := bucket.Object(name).NewWriter(ctx)

	_, err = io.Copy(objWriter, resp.Body)
	if err != nil {
		fmt.Println(err.Error())
	}
	objWriter.Close()
}

func updateWallpaper(ctx context.Context, ID string, data map[string]interface{}) (*firestore.WriteResult, error) {
	return firestoreClient.Collection(firestoreCollection).Doc(ID).Set(ctx, data, firestore.MergeAll)
}

func detectLabels(ctx context.Context, url string) ([]*vision2.EntityAnnotation, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	image, err := vision.NewImageFromReader(resp.Body)
	if err != nil {
		return nil, err
	}
	annotations, err := annoClient.DetectLabels(ctx, image, nil, 50)
	if err != nil {
		return nil, err
	}

	return annotations, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
