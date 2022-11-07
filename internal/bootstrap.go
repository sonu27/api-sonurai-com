package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"api/internal/client"
	"api/internal/middleware"
	"api/internal/service"
	"api/internal/updater"

	firebase "firebase.google.com/go"
	"github.com/go-chi/chi"
	rscors "github.com/rs/cors"
	"google.golang.org/api/option"
)

const collection = "BingWallpapers"

func Bootstrap() error {
	ctx := context.Background()

	saJSON, err := base64.StdEncoding.DecodeString(os.Getenv("FIRESTORE_SA"))
	if err != nil {
		return err
	}
	sa := option.WithCredentialsJSON(saJSON)

	firebaseClient, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return err
	}

	firestore, err := firebaseClient.Firestore(ctx)
	if err != nil {
		return err
	}

	wallpaperClient := client.NewClient(collection, firestore)

	svc := service.NewService(&wallpaperClient)

	cors := rscors.New(rscors.Options{
		AllowedOrigins:   []string{"https://sonurai.com", "http://localhost:3000"},
		AllowCredentials: true,
		Debug:            false,
	})

	r := chi.NewRouter()
	r.Use(cors.Handler)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(""))
	})
	r.Route("/wallpapers", func(r chi.Router) {
		r.Use(middleware.JSONHeaders)
		r.Get("/", svc.ListWallpapersHandler)
		r.Get("/tags/{tag}", svc.ListWallpapersByTagHandler)
		r.Get("/{id}", svc.GetWallpaperHandler)
	})

	u, err := updater.New(ctx, sa)
	if err != nil {
		return err
	}

	errs := make(chan error, 1)
	go func() {
		err := u.Start(ctx, sa)
		if err != nil {
			errs <- err
		}
		close(errs)
	}()

	go func() {
		port := os.Getenv("PORT")
		log.Printf("server started on http://localhost:%s", port)
		err := http.ListenAndServe(":"+port, r)
		if err != nil {
			errs <- err
		}
		close(errs)
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errs:
		return err
	case <-exit:
		return fmt.Errorf("sigterm received")
	}
}
