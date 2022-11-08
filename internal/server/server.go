package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"api/internal/middleware"
	"api/internal/store"

	"github.com/go-chi/chi"
	rscors "github.com/rs/cors"
)

func New(store store.Storer) Server {
	s := Server{store: store}

	cors := rscors.New(rscors.Options{
		AllowedOrigins:   []string{"https://sonurai.com", "http://localhost:3000"},
		AllowCredentials: true,
		Debug:            false,
	})

	r := chi.NewRouter()
	r.Use(cors.Handler)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(""))
	})
	r.Route("/wallpapers", func(r chi.Router) {
		r.Use(middleware.JSONHeaders)
		r.Get("/", s.ListWallpapersHandler)
		r.Get("/tags/{tag}", s.ListWallpapersByTagHandler)
		r.Get("/{id}", s.GetWallpaperHandler)
	})
	s.Handler = r
	return s
}

type Server struct {
	http.Handler
	store store.Storer
}

func (svc *Server) GetWallpaperHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var wallpaper *store.WallpaperWithTags
	if i, err := strconv.Atoi(id); err == nil {
		wallpaper, err = svc.store.GetByOldID(r.Context(), i)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		wallpaper, err = svc.store.Get(r.Context(), id)
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

func (svc *Server) ListWallpapersHandler(w http.ResponseWriter, r *http.Request) {
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

	data, err := svc.store.List(r.Context(), q)
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

func (svc *Server) ListWallpapersByTagHandler(w http.ResponseWriter, r *http.Request) {
	tag := chi.URLParam(r, "tag")
	var after float64 = 1

	if v := r.URL.Query().Get("after"); v != "" {
		if i, err := strconv.ParseFloat(v, 64); err == nil {
			after = i
		}
	}

	data, err := svc.store.ListByTag(r.Context(), tag, after)
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
