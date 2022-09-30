package internal

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"api/internal/client"
	"api/internal/server"
	"api/internal/service"
	"api/internal/updater"

	"github.com/allegro/bigcache"
	"github.com/go-chi/chi"
	rscors "github.com/rs/cors"
)

const collection = "BingWallpapers"

func Bootstrap() (err error) {
	ctx := context.Background()

	firebase, err := client.Firebase(ctx)
	if err != nil {
		return err
	}

	firestore, err := firebase.Firestore(ctx)
	if err != nil {
		return err
	}

	cache, err := bigcache.NewBigCache(bigcache.DefaultConfig(time.Hour * 24 * 30))
	if err != nil {
		return err
	}

	wallpaperClient := client.NewClient(collection, firestore)

	svc := service.NewService(cache, &wallpaperClient)

	cors := rscors.New(rscors.Options{
		AllowedOrigins:   []string{"https://sonurai.com", "http://localhost:3000"},
		AllowCredentials: true,
		Debug:            false,
	})

	r := chi.NewRouter()
	r.Use(cors.Handler)
	//r.Use(middleware.Logger)
	r.Use(server.WrapResponseWriter)
	r.Route("/wallpapers", func(r chi.Router) {
		r.Get("/", svc.ListWallpapersHandler)
		r.Get("/tags/{tag}", svc.ListWallpapersByTagHandler)
		r.Get("/{id}", svc.GetWallpaperHandler)
	})

	go updater.New(false, firebase, firestore)

	port := os.Getenv("PORT")
	log.Printf("server started on port %s", port)
	err = http.ListenAndServe(":"+port, r)

	return
}
