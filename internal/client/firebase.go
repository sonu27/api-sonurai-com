package client

import (
	"context"
	"encoding/base64"
	"os"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
)

func Firebase(ctx context.Context) (*firebase.App, error) {
	saJSON, err := base64.StdEncoding.DecodeString(os.Getenv("FIRESTORE_SA"))
	if err != nil {
		return nil, err
	}
	sa := option.WithCredentialsJSON(saJSON)

	return firebase.NewApp(ctx, nil, sa)
}
