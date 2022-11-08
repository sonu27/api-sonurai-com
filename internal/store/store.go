package store

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Storer interface {
	Get(ctx context.Context, id string) (*WallpaperWithTags, error)
	GetByOldID(ctx context.Context, id int) (*WallpaperWithTags, error)
	List(ctx context.Context, q ListQuery) (*ListResponse, error)
	ListByTag(ctx context.Context, tag string, after float64) (*ListResponse, error)
}

type ListQuery struct {
	Limit          int
	StartAfterDate int
	StartAfterID   string
	Reverse        bool
}

func New(collection string, firestore *firestore.Client) Store {
	return Store{
		collection: collection,
		firestore:  firestore,
	}
}

type Store struct {
	collection string
	firestore  *firestore.Client
}

func (s *Store) Get(ctx context.Context, id string) (*WallpaperWithTags, error) {
	doc, err := s.firestore.Collection(s.collection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

	wallpaper := new(WallpaperWithTags)
	if err := jsonToAny(doc.Data(), wallpaper); err != nil {
		return nil, err
	}

	return wallpaper, nil
}

func (s *Store) GetByOldID(ctx context.Context, id int) (*WallpaperWithTags, error) {
	iter := s.firestore.Collection(s.collection).Where("oldId", "==", id).Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if status.Code(err) == codes.NotFound || err == iterator.Done {
			return nil, nil
		}
		return nil, err
	}

	wallpaper := new(WallpaperWithTags)
	if err := jsonToAny(doc.Data(), &wallpaper); err != nil {
		return nil, err
	}

	return wallpaper, nil
}

func (s *Store) List(ctx context.Context, q ListQuery) (*ListResponse, error) {
	showPrev := false
	query := s.firestore.Collection(s.collection).Limit(q.Limit)

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

	var res ListResponse
	for _, v := range dsnap {
		var wallpaper Wallpaper
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
		res.Links = &Links{Next: fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s", last.Date, last.ID)}

		if showPrev {
			first := res.Data[0]
			res.Links.Prev = fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s&prev=1", first.Date, first.ID)
		}
	}

	return &res, nil
}

func (s *Store) ListByTag(ctx context.Context, tag string, after float64) (*ListResponse, error) {
	dsnap, err := s.firestore.Collection(s.collection).
		Where(fmt.Sprintf("tags.%s", tag), "<", after).
		Limit(36).
		OrderBy(fmt.Sprintf("tags.%s", tag), firestore.Desc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var res ListResponse
	for _, v := range dsnap {
		var wallpaper Wallpaper
		if err := jsonToAny(v.Data(), &wallpaper); err != nil {
			return nil, err
		}
		res.Data = append(res.Data, wallpaper)
	}

	if len(res.Data) > 0 {
		next := dsnap[len(dsnap)-1].Data()["tags"].(map[string]any)[tag].(float64)
		res.Links = &Links{Next: fmt.Sprintf("/wallpapers/tags/%s?after=%.16f", tag, next)}
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
