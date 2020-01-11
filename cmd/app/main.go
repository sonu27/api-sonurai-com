package main

import (
	"api/internal"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
	}

	if err := internal.Bootstrap(); err != nil {
		fmt.Fprint(os.Stderr, "Bootstrap error: "+err.Error())
	}
}
