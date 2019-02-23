package internal

import (
	"context"
	"encoding/json"
	"net/http"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"

	"github.com/go-chi/chi"
)

var (
	firestoreClient *firestore.Client
)

func listWallpapers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	iter := firestoreClient.Collection("wallpapers").
		OrderBy("date", firestore.Desc).
		Limit(10).
		Documents(ctx)

	docs := make(map[string][]map[string]interface{})

	var wallpapers []map[string]interface{}

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
		}

		wallpapers = append(wallpapers, doc.Data())
	}

	docs["data"] = wallpapers
	data, _ := json.Marshal(docs)

	w.Write(data)
}

func getWallpaper(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")
	dsnap, err := firestoreClient.Collection("wallpapers").Doc(name).Get(ctx)
	if err != nil {
	}
	m := dsnap.Data()

	data, _ := json.Marshal(m)

	w.Write(data)
}

func Bootstrap() error {
	ctx := context.Background()

	sa := option.WithCredentialsFile("serviceAccount.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return err
	}

	firestoreClient, err = app.Firestore(ctx)
	if err != nil {
		return err
	}
	defer firestoreClient.Close()

	r := chi.NewRouter()
	r.Get("/wallpapers", listWallpapers)
	r.Get("/wallpapers/{name}", getWallpaper)

	http.ListenAndServe(":8080", r)

	return nil
}
