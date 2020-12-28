package client

import (
	"api/internal/model"
	"cloud.google.com/go/firestore"
	"context"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type WallpaperClient interface {
	Get(ctx context.Context, id string) (map[string]interface{}, error)
	GetByOldID(ctx context.Context, id int) (map[string]interface{}, error)
	List(ctx context.Context, q ListQuery) (*model.ListResponse, error)
}

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

type ListQuery struct {
	Limit          int
	StartAfterDate int
	StartAfterID   string
	Reverse        bool
}

func (c *Client) Get(ctx context.Context, id string) (map[string]interface{}, error) {
	doc, err := c.firestore.Collection(c.collection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return doc.Data(), nil
}

func (c *Client) GetByOldID(ctx context.Context, id int) (map[string]interface{}, error) {
	iter := c.firestore.Collection(c.collection).Where("oldId", "==", id).Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if status.Code(err) == codes.NotFound || err == iterator.Done {
			return nil, nil
		} else {
			return nil, err
		}
	}

	return doc.Data(), nil
}

func (c *Client) List(ctx context.Context, q ListQuery) (*model.ListResponse, error) {
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

	iter := query.Documents(ctx)
	var res model.ListResponse
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var wallpaper model.ImageBasic
		mapstructure.Decode(doc.Data(), &wallpaper)
		res.Data = append(res.Data, wallpaper)
	}

	if q.Reverse {
		reverseImages(res.Data)
	}

	return &res, nil
}

func reverseImages(a []model.ImageBasic) {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}
