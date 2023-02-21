package firestore

import (
	"api/internal/updater/image"
	"cloud.google.com/go/firestore"
	"context"
	"encoding/json"
	firebase "firebase.google.com/go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewClient(ctx context.Context, collection string, app *firebase.App) (*Client, error) {
	firestore, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{collection: collection, firestore: firestore}, nil
}

type Client struct {
	collection string
	firestore  *firestore.Client
}

func (c *Client) Get(ctx context.Context, ID string) (*image.Image, error) {
	dsnap, err := c.firestore.Collection(c.collection).Doc(ID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

	b, err := json.Marshal(dsnap.Data())
	if err != nil {
		return nil, err
	}

	var result image.Image
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) Upsert(ctx context.Context, img image.Image) (*firestore.WriteResult, error) {
	var out map[string]any
	inrec, err := json.Marshal(img)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(inrec, &out); err != nil {
		return nil, err
	}

	return c.firestore.Collection(c.collection).Doc(img.ID).Set(ctx, out, firestore.MergeAll)
}
