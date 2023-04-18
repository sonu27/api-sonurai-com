package server

import (
	"api/internal/store"
	"api/view"
	"github.com/go-chi/chi"
	"net/http"
	"strconv"
)

func (s *server) AboutViewHandler(w http.ResponseWriter, r *http.Request) {
	type Page struct {
		Title string
	}

	if err := view.About.Execute(w, Page{
		Title: "About",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) ListWallpapersViewHandler(w http.ResponseWriter, r *http.Request) {
	q := store.ListQuery{Limit: 24}

	wallpapers, err := s.store.List(r.Context(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(wallpapers) == 0 {
		w.WriteHeader(404)
		return
	}

	type Page struct {
		Title      string
		Wallpapers []store.Wallpaper
	}

	if err := view.WallpaperIndex.Execute(w, Page{
		Title:      "Test",
		Wallpapers: wallpapers,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) GetWallpaperViewHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// todo: redirect to new id
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

	type Page struct {
		Title string
		W     *store.WallpaperWithTags
	}

	// todo: list tags by most relevant

	if err := view.WallpaperGet.Execute(w, Page{
		Title: wallpaper.Title,
		W:     wallpaper,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
