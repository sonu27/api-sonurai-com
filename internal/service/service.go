package service

import (
	"encoding/json"
	"fmt"
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

	b, _ = json.Marshal(*wallpaper)
	_ = svc.cache.Set(id, b)
	_, _ = w.Write(b)
}

func (svc *Service) ListWallpapersHandler(w http.ResponseWriter, r *http.Request) {
	q := client.ListQuery{Limit: 24}

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

	first := data.Data[0]
	last := data.Data[len(data.Data)-1]

	data.Links.Prev = fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s&prev=1", first.Date, first.ID)
	data.Links.Next = fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s", last.Date, last.ID)

	b, _ := json.Marshal(data)
	_, _ = w.Write(b)
}

func (svc *Service) ListWallpapersByTagHandler(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")

	data, err := svc.client.ListByTag(r.Context(), tag)
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
