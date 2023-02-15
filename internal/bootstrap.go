package internal

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"api/internal/server"
	"api/internal/store"
	"api/internal/updater"
	"api/internal/updater/pubsub"

	firebase "firebase.google.com/go"
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

	port := os.Getenv("PORT")
	srv := server.New(port, &wallpaperClient)

	u, err := updater.New()
	if err != nil {
		return err
	}

	errs := make(chan error, 1)
	go func() {
		err := pubsub.Start(ctx, os.Getenv("PROJECT_ID"), updater.TopicID, updater.SubID, u.Update)
		if err != nil {
			errs <- err
		}
		close(errs)
	}()

	go func() {
		log.Printf("server started on http://localhost:%s", port)
		err := srv.ListenAndServe()
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
		log.Println("server stopping")
		return nil
	}
}
