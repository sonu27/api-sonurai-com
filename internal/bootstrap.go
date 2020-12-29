package internal

import (
	"api/internal/client"
	"api/internal/server"
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
	firestoreClient, err := client.FirestoreClient(ctx)
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
	r.Use(server.WrapResponseWriter)
	r.Route("/wallpapers", func(r chi.Router) {
		r.Get("/", svc.ListWallpapersHandler)
		r.Get("/{id}", svc.GetWallpaperHandler)
	})

	port := os.Getenv("PORT")
	log.Printf("server started on port %s", port)
	err = http.ListenAndServe(":"+port, r)

	return
}
