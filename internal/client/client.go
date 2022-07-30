package client

import (
	"context"
	"encoding/json"
	"fmt"

	"api/internal/model"
	"api/internal/service"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewClient(collection string, firestore *firestore.Client) *Client {
	return &Client{
		collection: collection,
		firestore:  firestore,
	}
}

type Client struct {
	collection string
	firestore  *firestore.Client
}

func (c *Client) Get(ctx context.Context, id string) (*model.WallpaperWithTags, error) {
	doc, err := c.firestore.Collection(c.collection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

	wallpaper := new(model.WallpaperWithTags)
	if err := jsonToInterface(doc.Data(), &wallpaper); err != nil {
		return nil, err
	}

	return wallpaper, nil
}

func (c *Client) GetByOldID(ctx context.Context, id int) (*model.WallpaperWithTags, error) {
	iter := c.firestore.Collection(c.collection).Where("oldId", "==", id).Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if status.Code(err) == codes.NotFound || err == iterator.Done {
			return nil, nil
		}
		return nil, err
	}

	wallpaper := new(model.WallpaperWithTags)
	if err := jsonToInterface(doc.Data(), &wallpaper); err != nil {
		return nil, err
	}

	return wallpaper, nil
}

func (c *Client) List(ctx context.Context, q service.ListQuery) (*model.ListResponse, error) {
	query := c.firestore.Collection(c.collection).Limit(q.Limit)

	if q.Reverse {
		query = query.
			OrderBy("date", firestore.Asc).
			OrderBy("id", firestore.Desc)

	} else {
		query = query.
			OrderBy("date", firestore.Desc).
			OrderBy("id", firestore.Asc)
	}

	if q.StartAfterDate != 0 && q.StartAfterID != "" {
		query = query.StartAfter(q.StartAfterDate, q.StartAfterID)
	}

	dsnap, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var res model.ListResponse
	for _, v := range dsnap {
		var wallpaper model.Wallpaper
		if err := jsonToInterface(v.Data(), &wallpaper); err != nil {
			return nil, err
		}
		res.Data = append(res.Data, wallpaper)
	}

	if q.Reverse {
		reverse(res.Data)
	}

	return &res, nil
}

func (c *Client) ListByTag(ctx context.Context, tag string) (*model.ListResponse, error) {
	dsnap, err := c.firestore.Collection(c.collection).
		Limit(36).
		OrderBy(fmt.Sprintf("tags.%s", tag), firestore.Desc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var res model.ListResponse
	for _, v := range dsnap {
		var wallpaper model.Wallpaper
		if err := jsonToInterface(v.Data(), &wallpaper); err != nil {
			return nil, err
		}
		res.Data = append(res.Data, wallpaper)
	}

	return &res, nil
}

func jsonToInterface(in map[string]any, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, out); err != nil {
		return err
	}
	return nil
}

func reverse[T any](a []T) {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}
