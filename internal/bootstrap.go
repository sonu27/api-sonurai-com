package internal

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/allegro/bigcache"
	"github.com/go-chi/chi"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/cors"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	firestoreCollection = "BingWallpapers"
)

var (
	firestoreClient *firestore.Client
	cache           *bigcache.BigCache
)

type ListResponse struct {
	Data []ImageBasic `json:"data"`
}

type ImageBasic struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Copyright string `json:"copyright"`
	Date      int    `json:"date"`
	Filename  string `json:"filename"`
	Market    string `json:"market"`
}

type Image struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	Copyright        string            `json:"copyright"`
	Date             int               `json:"date"`
	Filename         string            `json:"filename"`
	Market           string            `json:"market"`
	LabelAnnotations []LabelAnnotation `json:"labelAnnotations,omitempty"`
}

type LabelAnnotation struct {
	Description string  `json:"description"`
	MID         string  `json:"mid"`
	Score       float64 `json:"score"`
	Topicality  float64 `json:"topicality"`
}

func Bootstrap() error {
	ctx := context.Background()
	firestoreClient = GetFirestoreClient(ctx)
	cache, _ = bigcache.NewBigCache(bigcache.DefaultConfig(24 * time.Hour))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://sonurai.com", "http://localhost:3000"},
		AllowCredentials: true,
		Debug:            false,
	})

	r := chi.NewRouter()
	r.Use(c.Handler)
	r.Get("/wallpapers", listWallpapersHandler)
	r.Get("/wallpapers/{id}", getWallpaperHandler)

	port := os.Getenv("PORT")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))

	return nil
}

func getWallpaperHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	id := chi.URLParam(r, "id")
	b, _ := cache.Get(id)
	if len(b) > 0 {
		etag := fmt.Sprintf("\"%x\"", md5.Sum(b))
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", secondsExpiresIn()))
		w.Header().Set("ETag", etag)

		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(304)
			return
		}

		w.Write(b)
		return
	}

	ctx := r.Context()

	if isNumeric(id) {
		getByOldId(ctx, w, r, id)
		return
	}

	dsnap, err := firestoreClient.Collection(firestoreCollection).Doc(id).Get(ctx)
	if status.Code(err) == codes.NotFound {
		w.WriteHeader(404)
		return
	}

	if dsnap.Exists() {
		data := dsnap.Data()

		outputAndCache(w, r, id, data)
		return
	}

	w.WriteHeader(404)
}

func getByOldId(ctx context.Context, w http.ResponseWriter, r *http.Request, oldId string) {
	i, _ := strconv.Atoi(oldId)
	iter := firestoreClient.Collection(firestoreCollection).Where("oldId", "==", i).Documents(ctx)

	doc, err := iter.Next()
	if err == iterator.Done {
		w.WriteHeader(404)
		return
	}

	if status.Code(err) == codes.NotFound {
		w.WriteHeader(404)
		return
	}

	if err != nil {
		w.WriteHeader(500)
		return
	}

	outputAndCache(w, r, oldId, doc.Data())
}

func outputAndCache(w http.ResponseWriter, r *http.Request, id string, data map[string]interface{}) {
	var result Image
	mapstructure.Decode(data, &result)
	b, _ := json.Marshal(result)
	cache.Set(id, b)

	etag := fmt.Sprintf("\"%x\"", md5.Sum(b))
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", secondsExpiresIn()))
	w.Header().Set("ETag", etag)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(304)
		return
	}

	w.Write(b)
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func listWallpapersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	iter := new(firestore.DocumentIterator)

	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			offset = i
		}
	}

	iter = firestoreClient.Collection(firestoreCollection).
		OrderBy("date", firestore.Desc).
		OrderBy("id", firestore.Asc).
		Offset(offset).
		Limit(10).
		Documents(ctx)

	startAfterDate := 0
	if v := r.URL.Query().Get("startAfterDate"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			startAfterDate = i
		}
	}

	startAfterID := ""
	if v := r.URL.Query().Get("startAfterID"); v != "" {
		startAfterID = v
	}

	if startAfterDate != 0 && startAfterID != "" {
		iter = firestoreClient.Collection(firestoreCollection).
			OrderBy("date", firestore.Desc).
			OrderBy("id", firestore.Asc).
			StartAfter(startAfterDate, startAfterID).
			Limit(10).
			Documents(ctx)
	}

	var res ListResponse
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		var wallpaper ImageBasic
		mapstructure.Decode(doc.Data(), &wallpaper)
		res.Data = append(res.Data, wallpaper)
	}

	b, _ := json.Marshal(res)
	etag := fmt.Sprintf("\"%x\"", md5.Sum(b))
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", secondsExpiresIn()))
	w.Header().Set("ETag", etag)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(304)
		return
	}

	w.Write(b)
}

func GetFirestoreClient(ctx context.Context) *firestore.Client {
	saJSON, err := base64.StdEncoding.DecodeString(os.Getenv("FIRESTORE_SA"))
	if err != nil {
		panic(err)
	}
	sa := option.WithCredentialsJSON(saJSON)
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		panic(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		panic(err)
	}

	return client
}

func secondsExpiresIn() int {
	now := time.Now()
	expireTime := time.Date(now.Year(), now.Month(), now.Day(), 8, 5, 0, 0, time.UTC)
	secsInDay := 86400

	var secondsExpiresIn int
	if now.Before(expireTime) {
		diff := expireTime.Sub(now)
		secondsExpiresIn = int(diff.Seconds())
	} else {
		diff := now.Sub(expireTime)
		secondsExpiresIn = secsInDay - int(diff.Seconds())
	}

	return secondsExpiresIn
}
