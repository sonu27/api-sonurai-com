package server

import (
	"api/internal/store"
	"api/view"
	"fmt"
	"github.com/go-chi/chi"
	"net/http"
	"strconv"
)

func (s *server) AboutViewHandler(w http.ResponseWriter, r *http.Request) {
	if err := view.About.Execute(w, nil); err != nil {
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
		Wallpapers []store.Wallpaper
	}

	if err := view.WallpaperIndex.Execute(w, Page{
		Wallpapers: wallpapers,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) ListWallpapersByTagViewHandler(w http.ResponseWriter, r *http.Request) {
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

	type Page struct {
		Tag        string
		Wallpapers []store.Wallpaper
	}

	if err := view.WallpaperListByTag.Execute(w, Page{
		Tag:        tag,
		Wallpapers: wallpapers,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) GetWallpaperViewHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if i, err := strconv.Atoi(id); err == nil {
		wallpaper, err := s.store.GetByOldID(r.Context(), i)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if wallpaper == nil {
			w.WriteHeader(404)
			return
		}

		http.Redirect(w, r, "/bingwallpapers/"+wallpaper.ID, http.StatusMovedPermanently)
		return
	}

	wallpaper, err := s.store.Get(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if wallpaper == nil {
		w.WriteHeader(404)
		return
	}

	type Page struct {
		W *store.WallpaperWithTags
	}

	// todo: list tags by most relevant

	if err := view.WallpaperGet.Execute(w, Page{
		W: wallpaper,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
