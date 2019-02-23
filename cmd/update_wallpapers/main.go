package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/joho/godotenv"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
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

	sa := option.WithCredentialsFile("serviceAccount.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()

	resp, _ := http.Get("https://www.bing.com/HPImageArchive.aspx?format=js&n=10&mbl=1&mkt=en-ww")
	defer resp.Body.Close()

	var v map[string][]map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&v)
	n := v["images"]

	for _, v := range n {
		name := strings.Replace(v["urlbase"].(string), "/az/hprichbg/rb/", "", 1)
		name = strings.Replace(name, "/th?id=OHR.", "", 1)
		x := strings.Split(name, "_")
		name = x[0]
		_, err = client.Collection("wallpapers").Doc(name).Set(ctx, map[string]interface{}{
			"title": v["copyright"],
			"name":  name,
			"date":  v["startdate"],
		})
		if err != nil {
			log.Fatalf("Failed adding: %v", err)
		}
	}

	iter := client.Collection("wallpapers").OrderBy("date", firestore.Desc).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		fmt.Println(doc.Data())
	}

	return nil
}
