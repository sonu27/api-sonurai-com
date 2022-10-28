package main

import (
	"log"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := Bootstrap(); err != nil {
		log.Fatal("bootstrap error", err)
	}
}

func Bootstrap() (err error) {
	return nil
	//ctx := context.Background()

	//firebase, err := client.Firebase(ctx)
	//if err != nil {
	//	return err
	//}
	//
	//firestore, err := firebase.Firestore(ctx)
	//if err != nil {
	//	return err
	//}

	//return updater.New(true, firebase, firestore)
}
