package main

import (
	"api/internal"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	err := internal.Bootstrap()
	if err != nil {
		fmt.Fprint(os.Stderr, "Bootstrap error: "+err.Error())
	}
}
