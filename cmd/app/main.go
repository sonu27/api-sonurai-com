package main

import (
	"api/internal"
	"fmt"
	"os"
)

func main() {
	err := internal.Bootstrap()
	if err != nil {
		fmt.Fprint(os.Stderr, "Bootstrap error: "+err.Error())
	}
}
