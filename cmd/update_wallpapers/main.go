package main

import (
	"api/internal"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	err := Bootstrap()
	if err != nil {
		fmt.Fprint(os.Stderr, "Bootstrap error: "+err.Error())
	}
}

func Bootstrap() error {
	ctx := context.Background()

	markets := []string{
		"en-ww",
		"en-gb",
		"en-us",
		"zh-cn",
	}

	for _, v := range markets {
		updateWallpapers(ctx, v)
	}

	return nil
}

func updateWallpapers(ctx context.Context, market string) {
	firestoreClient := internal.GetFirestoreClient(ctx)
	defer firestoreClient.Close()

	resp, _ := http.Get("https://www.bing.com/HPImageArchive.aspx?format=js&n=10&mbl=1&mkt=" + market)
	defer resp.Body.Close()

	var v map[string][]map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&v)
	n := v["images"]

	for _, v := range n {
		name := strings.Replace(v["urlbase"].(string), "/az/hprichbg/rb/", "", 1)
		name = strings.Replace(name, "/th?id=OHR.", "", 1)
		x := strings.Split(name, "_")
		name = x[0]

		_, err := firestoreClient.Collection("wallpapers").Doc(name).Set(ctx, map[string]interface{}{
			"title": v["copyright"],
			"name":  name,
			"date":  v["startdate"],
		})
		if err != nil {
			log.Fatalf("Failed adding: %v", err)
		}
	}
}
