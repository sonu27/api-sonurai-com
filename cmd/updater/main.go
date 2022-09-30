package main

import (
	"context"
	"log"

	"api/internal/client"
	"api/internal/update"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := Bootstrap(); err != nil {
		log.Fatal("bootstrap error", err)
	}
}

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

	return update.New(true, firebase, firestore)
}
