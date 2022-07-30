package service

import (
	"encoding/json"
	"net/http"
	"strconv"

	"api/internal/client"
	"api/internal/model"

	"github.com/go-chi/chi"
)

type Cache interface {
	Get(key string) ([]byte, error)
	Set(key string, entry []byte) error
}

func NewService(cache Cache, client client.WallpaperClient) *Service {
	return &Service{
		cache:  cache,
		client: client,
	}
}

type Service struct {
	cache  Cache
	client client.WallpaperClient
}

func (svc *Service) GetWallpaperHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	b, _ := svc.cache.Get(id)
	if len(b) > 0 {
		_, _ = w.Write(b)
		return
	}

	var image *model.Image
	if i, err := strconv.Atoi(id); err == nil {
		image, err = svc.client.GetByOldID(r.Context(), i)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		image, err = svc.client.Get(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if image == nil {
		w.WriteHeader(404)
		return
	}

	b, _ = json.Marshal(*image)
	_ = svc.cache.Set(id, b)
	_, _ = w.Write(b)
}

func (svc *Service) ListWallpapersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := client.ListQuery{
		Limit:          24,
		StartAfterDate: 0,
		StartAfterID:   "",
		Reverse:        false,
	}

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

	data, err := svc.client.List(ctx, q)
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
	ctx := r.Context()

	data, err := svc.client.ListByTag(ctx, tag)
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
