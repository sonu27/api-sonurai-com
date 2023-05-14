package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"api/internal/middleware"
	"api/internal/store"
	"github.com/go-chi/chi/v5"
	rscors "github.com/rs/cors"
)

func New(port string, store store.Storer) http.Server {
	cors := rscors.New(rscors.Options{
		AllowedOrigins: []string{
			"https://sonurai.com",
			"https://*.vercel.app",
			"http://localhost:3000",
		},
		AllowCredentials: true,
		Debug:            false,
	})

	s := server{store: store}
	r := chi.NewRouter()
	r.Use(cors.Handler)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(""))
	})
	r.Route("/wallpapers", func(r chi.Router) {
		r.Use(middleware.JSONContentType)
		r.Get("/", s.ListWallpapersHandler)
		r.Get("/tags/{tag}", s.ListWallpapersByTagHandler)
		r.Get("/tags", s.ListTagsHandler)
		r.Get("/{id}", s.GetWallpaperHandler)
	})

	return http.Server{
		Addr:        ":" + port,
		Handler:     r,
		ReadTimeout: time.Second * 10,
	}
}

type server struct {
	store store.Storer
}

func (s *server) GetWallpaperHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var wallpaper *store.WallpaperWithTags
	if i, err := strconv.Atoi(id); err == nil {
		wallpaper, err = s.store.GetByOldID(r.Context(), i)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		wallpaper, err = s.store.Get(r.Context(), id)
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

func (s *server) ListWallpapersHandler(w http.ResponseWriter, r *http.Request) {
	showPrev := false
	q := store.ListQuery{Limit: 24}

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

	if q.StartAfterDate != 0 && q.StartAfterID != "" {
		showPrev = true
	}

	wallpapers, err := s.store.List(r.Context(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(wallpapers) == 0 {
		w.WriteHeader(404)
		return
	}

	res := ListResponse{
		Data: wallpapers,
	}

	last := wallpapers[len(wallpapers)-1]
	res.Links = &Links{Next: fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s", last.Date, last.ID)}

	if showPrev {
		first := wallpapers[0]
		res.Links.Prev = fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s&prev=1", first.Date, first.ID)
	}

	b, _ := json.Marshal(res)
	_, _ = w.Write(b)
}

func (s *server) ListWallpapersByTagHandler(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	var after float64 = 1

	if v := r.URL.Query().Get("after"); v != "" {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			after = i
		}
	}

	wallpapers, next, err := s.store.ListByTag(r.Context(), tag, after)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(wallpapers) == 0 {
		w.WriteHeader(404)
		return
	}

	res := ListResponse{
		Data: wallpapers,
	}

	if len(wallpapers) == 36 && next > 0 {
		res.Links = &Links{Next: fmt.Sprintf("/wallpapers/tags/%s?after=%.16f", tag, next)}
	}

	b, _ := json.Marshal(res)
	_, _ = w.Write(b)
}

func (s *server) ListTagsHandler(w http.ResponseWriter, r *http.Request) {
	tags, err := s.store.GetTags(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b, _ := json.Marshal(tags)
	_, _ = w.Write(b)
}
