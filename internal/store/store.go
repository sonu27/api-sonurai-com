package store

import (
	"context"
	"encoding/json"
	"errors"
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
	GetTags(ctx context.Context) (map[string]int, error)
	List(ctx context.Context, q ListQuery) ([]Wallpaper, error)
	ListByTag(ctx context.Context, tag string, after float64) ([]Wallpaper, float64, error)
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
	if err := convertMapTo(doc.Data(), wallpaper); err != nil {
		return nil, err
	}

	return wallpaper, nil
}

func (s *Store) GetByOldID(ctx context.Context, id int) (*WallpaperWithTags, error) {
	iter := s.firestore.Collection(s.collection).Where("oldId", "==", id).Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if status.Code(err) == codes.NotFound || errors.Is(err, iterator.Done) {
			return nil, nil
		}
		return nil, err
	}

	wallpaper := new(WallpaperWithTags)
	if err := convertMapTo(doc.Data(), &wallpaper); err != nil {
		return nil, err
	}

	return wallpaper, nil
}

func (s *Store) GetTags(ctx context.Context) (map[string]int, error) {
	doc, err := s.firestore.Collection("tags").Doc("popular").Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}

	tags := make(map[string]int)
	if err := convertMapTo(doc.Data(), &tags); err != nil {
		return nil, err
	}

	return tags, nil
}

func (s *Store) List(ctx context.Context, q ListQuery) ([]Wallpaper, error) {
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
		query = query.StartAfter(q.StartAfterDate, q.StartAfterID)
	}

	dsnap, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	wallpapers := make([]Wallpaper, 0, len(dsnap))
	for _, v := range dsnap {
		var wallpaper Wallpaper
		if err := convertMapTo(v.Data(), &wallpaper); err != nil {
			return nil, err
		}
		wallpapers = append(wallpapers, wallpaper)
	}

	if q.Reverse {
		reverse(wallpapers)
	}

	return wallpapers, nil
}

func (s *Store) ListByTag(ctx context.Context, tag string, after float64) ([]Wallpaper, float64, error) {
	dsnap, err := s.firestore.Collection(s.collection).
		Where(fmt.Sprintf("tags.%s", tag), "<", after).
		Limit(36).
		OrderBy(fmt.Sprintf("tags.%s", tag), firestore.Desc).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, 0, err
	}

	wallpapers := make([]Wallpaper, 0, len(dsnap))
	for _, v := range dsnap {
		var wallpaper Wallpaper
		if err := convertMapTo(v.Data(), &wallpaper); err != nil {
			return nil, 0, err
		}
		wallpapers = append(wallpapers, wallpaper)
	}

	var next float64
	if len(wallpapers) > 0 {
		next = dsnap[len(dsnap)-1].Data()["tags"].(map[string]any)[tag].(float64)
	}

	return wallpapers, next, nil
}

func convertMapTo(in map[string]any, out any) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func reverse[T comparable](s []T) {
	sort.SliceStable(s, func(i, j int) bool {
		return i > j
	})
}
