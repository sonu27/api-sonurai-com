package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"api/internal/server"
	"api/internal/store"
	"api/internal/updater"
	"api/internal/updater/pubsub"

	firebase "firebase.google.com/go"
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

	wallpaperClient := store.New(collection, firestore)

	port := os.Getenv("PORT")
	srv := server.New(port, &wallpaperClient)

	u, err := updater.New(ctx, sa)
	if err != nil {
		return err
	}

	errs := make(chan error, 1)
	go func() {
		err := pubsub.Start(ctx, sa, updater.ProjectID, updater.TopicID, u.Update)
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
		return fmt.Errorf("sigterm received")
	}
}
