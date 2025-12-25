package internal

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	firebase "firebase.google.com/go"

	"api/internal/api"
	"api/internal/handler"
	"api/internal/server"
	"api/internal/store"
	"api/internal/updater"
	"api/internal/updater/pubsub"
)

const collection = "BingWallpapers"

func Bootstrap() error {
	ctx := context.Background()

	conf := &firebase.Config{ProjectID: os.Getenv("PROJECT_ID")}
	firebaseClient, err := firebase.NewApp(ctx, conf)
	if err != nil {
		return err
	}

	firestore, err := firebaseClient.Firestore(ctx)
	if err != nil {
		return err
	}

	wallpaperClient := store.New(collection, firestore)

	hh := handler.New(&wallpaperClient)
	h, err := api.NewServer(hh)
	if err != nil {
		return err
	}

	port := os.Getenv("PORT")

	srv := server.New(port, h)

	u, err := updater.New()
	if err != nil {
		return err
	}

	errs := make(chan error, 2)
	go func() {
		if err := pubsub.Start(ctx, os.Getenv("PROJECT_ID"), updater.TopicID, updater.SubID, u.Update); err != nil {
			errs <- err
		}
	}()
	go func() {
		log.Printf("server started on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil {
			errs <- err
		}
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errs:
		// Attempt graceful shutdown on error
		_ = srv.Shutdown(ctx)
		return err
	case <-exit:
		log.Println("shutdown signal received")
		if err := srv.Shutdown(ctx); err != nil {
			return err
		}
		log.Println("server stopped")
		return nil
	}
}
