package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"api/internal/model"
	"api/internal/service"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewClient(collection string, firestore *firestore.Client) Client {
	return Client{
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
	if err := jsonToAny(doc.Data(), wallpaper); err != nil {
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
	if err := jsonToAny(doc.Data(), &wallpaper); err != nil {
		return nil, err
	}

	return wallpaper, nil
}

func (c *Client) List(ctx context.Context, q service.ListQuery) (*model.ListResponse, error) {
	showPrev := false
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
		showPrev = true
		query = query.StartAfter(q.StartAfterDate, q.StartAfterID)
	}

	dsnap, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var res model.ListResponse
	for _, v := range dsnap {
		var wallpaper model.Wallpaper
		if err := jsonToAny(v.Data(), &wallpaper); err != nil {
			return nil, err
		}
		res.Data = append(res.Data, wallpaper)
	}

	if q.Reverse {
		reverse(res.Data)
	}

	if len(res.Data) > 0 {
		last := res.Data[len(res.Data)-1]
		res.Links = &model.Links{Next: fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s", last.Date, last.ID)}

		if showPrev {
			first := res.Data[0]
			res.Links.Prev = fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s&prev=1", first.Date, first.ID)
		}
	}

	return &res, nil
}

func (c *Client) ListByTag(ctx context.Context, tag string, after float64) (*model.ListResponse, error) {
	dsnap, err := c.firestore.Collection(c.collection).
		Where(fmt.Sprintf("tags.%s", tag), "<", after).
		Limit(36).
		OrderBy(fmt.Sprintf("tags.%s", tag), firestore.Desc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var res model.ListResponse
	for _, v := range dsnap {
		var wallpaper model.Wallpaper
		if err := jsonToAny(v.Data(), &wallpaper); err != nil {
			return nil, err
		}
		res.Data = append(res.Data, wallpaper)
	}

	if len(res.Data) > 0 {
		next := dsnap[len(dsnap)-1].Data()["tags"].(map[string]any)[tag].(float64)
		res.Links = &model.Links{Next: fmt.Sprintf("/wallpapers/tags/%s?after=%.16f", tag, next)}
	}

	return &res, nil
}

func jsonToAny(in map[string]any, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, out); err != nil {
		return err
	}
	return nil
}

func reverse[T comparable](s []T) {
	sort.SliceStable(s, func(i, j int) bool {
		return i > j
	})
}
