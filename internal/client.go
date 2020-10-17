package internal

import (
	"context"
	"encoding/base64"
	"os"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

const (
	firestoreCollection = "BingWallpapers"
)

func GetFirestoreClient(ctx context.Context) (*firestore.Client, error) {
	saJSON, err := base64.StdEncoding.DecodeString(os.Getenv("FIRESTORE_SA"))
	if err != nil {
		return nil, err
	}
	sa := option.WithCredentialsJSON(saJSON)
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	return client, nil
}
