package main

import (
	"log"

	"api/internal"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := internal.Bootstrap(); err != nil {
		log.Fatal("bootstrap error", err)
	}
}
