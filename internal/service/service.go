package service

import (
	"api/internal/client"
	"api/internal/model"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi"
	"github.com/mitchellh/mapstructure"
	"net/http"
	"strconv"
	"time"
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
		etag := fmt.Sprintf("\"%x\"", md5.Sum(b))
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", secondsExpiresIn()))
		w.Header().Set("ETag", etag)

		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(304)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		_, _ = w.Write(b)
		return
	}

	data, err := svc.client.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if data == nil {
		w.WriteHeader(404)
		return
	}

	svc.outputAndCache(w, r, id, data)
}

func (svc *Service) GetOldWallpaperHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	b, _ := svc.cache.Get(id)
	if len(b) > 0 {
		etag := fmt.Sprintf("\"%x\"", md5.Sum(b))
		w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", secondsExpiresIn()))
		w.Header().Set("ETag", etag)

		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(304)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		_, _ = w.Write(b)
		return
	}

	i, _ := strconv.Atoi(id)
	data, err := svc.client.GetByOldID(r.Context(), i)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if data == nil {
		w.WriteHeader(404)
		return
	}

	svc.outputAndCache(w, r, id, data)
}

func (svc *Service) ListWallpapersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	q := client.ListQuery{
		Limit:          10,
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

	data, err := svc.client.List(ctx, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	b, _ := json.Marshal(data)
	etag := fmt.Sprintf("\"%x\"", md5.Sum(b))
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", secondsExpiresIn()))
	w.Header().Set("ETag", etag)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(304)
		return
	}

	_, _ = w.Write(b)
}

func (svc *Service) outputAndCache(w http.ResponseWriter, r *http.Request, id string, data map[string]interface{}) {
	var result model.Image
	_ = mapstructure.Decode(data, &result)
	b, _ := json.Marshal(result)
	_ = svc.cache.Set(id, b)

	etag := fmt.Sprintf("\"%x\"", md5.Sum(b))
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", secondsExpiresIn()))
	w.Header().Set("ETag", etag)

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(304)
		return
	}

	_, _ = w.Write(b)
}

func secondsExpiresIn() int {
	now := time.Now()
	expireTime := time.Date(now.Year(), now.Month(), now.Day(), 8, 5, 0, 0, time.UTC)
	secsInDay := 86400

	var secondsExpiresIn int
	if now.Before(expireTime) {
		diff := expireTime.Sub(now)
		secondsExpiresIn = int(diff.Seconds())
	} else {
		diff := now.Sub(expireTime)
		secondsExpiresIn = secsInDay - int(diff.Seconds())
	}

	return secondsExpiresIn
}
