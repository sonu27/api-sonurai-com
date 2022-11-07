package service

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"api/internal/model"

	"github.com/go-chi/chi"
)

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

func NewService(client WallpaperClient) Service {
	return Service{
		client: client,
	}
}

type Service struct {
	client WallpaperClient
}

func (svc *Service) GetWallpaperHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

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
	_, _ = w.Write(b)
}
