package internal

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/allegro/bigcache"
	"github.com/go-chi/chi"
	"github.com/rs/cors"
)

var (
	firestoreClient *firestore.Client
	cache           *bigcache.BigCache
)

func Bootstrap() (err error) {
	ctx := context.Background()
	if firestoreClient, err = GetFirestoreClient(ctx); err != nil {
		return
	}
	if cache, err = bigcache.NewBigCache(bigcache.DefaultConfig(24 * time.Hour)); err != nil {
		return
	}

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://sonurai.com", "http://localhost:3000"},
		AllowCredentials: true,
		Debug:            false,
	})

	r := chi.NewRouter()
	r.Use(c.Handler)
	//r.Use(middleware.Logger)
	r.Get("/wallpapers", listWallpapersHandler)
	r.Get("/wallpapers/{id}", getWallpaperHandler)
	r.Get("/wallpapers/{id:[\\d]+}", getWallpaperHandlerLegacy)

	port := os.Getenv("PORT")
	log.Printf("server started on port %s", port)
	err = http.ListenAndServe(":"+port, r)

	return
}
