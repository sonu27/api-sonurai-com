package main

import (
	"api/internal"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	if err := internal.Bootstrap(); err != nil {
		log.Fatal("bootstrap error", err)
	}
}
