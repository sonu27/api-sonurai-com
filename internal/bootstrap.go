package internal

import (
	"api/internal/client"
	"api/internal/service"
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/allegro/bigcache"
	"github.com/go-chi/chi"
	"github.com/rs/cors"
)

const collection = "BingWallpapers"

func Bootstrap() (err error) {
	ctx := context.Background()
	firestoreClient, err := getFirestoreClient(ctx)
	if err != nil {
		return
	}
	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(24 * time.Hour))
	if err != nil {
		return
	}

	wallpaperClient := client.NewClient(collection, firestoreClient)
	svc := service.NewService(cache, wallpaperClient)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://sonurai.com", "http://localhost:3000"},
		AllowCredentials: true,
		Debug:            false,
	})

	r := chi.NewRouter()
	r.Use(c.Handler)
	//r.Use(middleware.Logger)
	r.Get("/wallpapers", svc.ListWallpapersHandler)
	r.Get("/wallpapers/{id}", svc.GetWallpaperHandler)
	r.Get("/wallpapers/{id:[\\d]+}", svc.GetOldWallpaperHandler)

	port := os.Getenv("PORT")
	log.Printf("server started on port %s", port)
	err = http.ListenAndServe(":"+port, r)

	return
}
