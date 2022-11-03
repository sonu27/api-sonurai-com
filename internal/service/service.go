package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"api/internal/model"

	"github.com/go-chi/chi"
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, entry []byte) error
}

type WallpaperClient interface {
	Get(ctx context.Context, id string) (*model.WallpaperWithTags, error)
	GetByOldID(ctx context.Context, id int) (*model.WallpaperWithTags, error)
	List(ctx context.Context, q ListQuery) (*model.ListResponse, error)
	ListByTag(ctx context.Context, tag string, after float64) (*model.ListResponse, error)
}

type ListQuery struct {
	Limit          int
	StartAfterDate int
	StartAfterID   string
	Reverse        bool
}

func NewService(cache Cache, client WallpaperClient) Service {
	return Service{
		cache:  cache,
		client: client,
	}
}

type Service struct {
	cache  Cache
	client WallpaperClient
}

func (svc *Service) GetWallpaperHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cacheKey := "id_" + id

	if b, _ := svc.cache.Get(cacheKey); len(b) > 0 {
		_, _ = w.Write(b)
		return
	}

	var wallpaper *model.WallpaperWithTags
	if i, err := strconv.Atoi(id); err == nil {
		wallpaper, err = svc.client.GetByOldID(r.Context(), i)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		wallpaper, err = svc.client.Get(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if wallpaper == nil {
		w.WriteHeader(404)
		return
	}

	b, _ := json.Marshal(wallpaper)
	_ = svc.cache.Set(cacheKey, b)
	_, _ = w.Write(b)
}

func (svc *Service) ListWallpapersHandler(w http.ResponseWriter, r *http.Request) {
	q := ListQuery{Limit: 24}

	if v := r.URL.Query().Get("startAfterDate"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			q.StartAfterDate = i
		}
	}

	if v := r.URL.Query().Get("startAfterID"); v != "" {
		q.StartAfterID = v
	}

	if v := r.URL.Query().Get("prev"); v != "" {
		q.Reverse = true
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil && i > 0 && i < q.Limit {
			q.Limit = i
		}
	}

	var sb strings.Builder
	sb.WriteString(string(rune(q.Limit)))
	sb.WriteString(q.StartAfterID)
	sb.WriteString(string(rune(q.StartAfterDate)))
	if q.Reverse {
		sb.WriteString("reverse")
	}
	cacheKey := sb.String()

	if b, _ := svc.cache.Get(cacheKey); len(b) > 0 {
		_, _ = w.Write(b)
		return
	}

	data, err := svc.client.List(r.Context(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(data.Data) == 0 {
		w.WriteHeader(404)
		return
	}

	b, _ := json.Marshal(data)
	_ = svc.cache.Set(cacheKey, b)
	_, _ = w.Write(b)
}

func (svc *Service) ListWallpapersByTagHandler(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	var after float64 = 1

	if v := r.URL.Query().Get("after"); v != "" {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			after = i
		}
	}

	cacheKey := fmt.Sprintf("tag_%s_%.16f", tag, after)

	if b, _ := svc.cache.Get(cacheKey); len(b) > 0 {
		_, _ = w.Write(b)
		return
	}

	data, err := svc.client.ListByTag(r.Context(), tag, after)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(data.Data) == 0 {
		w.WriteHeader(404)
		return
	}

	b, _ := json.Marshal(data)
	_ = svc.cache.Set(cacheKey, b)
	_, _ = w.Write(b)
}
