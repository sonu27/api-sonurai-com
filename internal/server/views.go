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
	showPrev := false
	q := store.ListQuery{Limit: 24}

	chi.URLParam(r, "date")

	if v := chi.URLParam(r, "date"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			q.StartAfterDate = i
		}
	}

	if v := chi.URLParam(r, "id"); v != "" {
		q.StartAfterID = v
	}

	if v := r.URL.Query().Get("prev"); v != "" {
		q.Reverse = true
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

	type Page struct {
		Prev       string
		Next       string
		Wallpapers []store.Wallpaper
	}

	p := Page{
		Next:       fmt.Sprintf("/bingwallpapers/page/%d/%s", wallpapers[len(wallpapers)-1].Date, wallpapers[len(wallpapers)-1].ID),
		Wallpapers: wallpapers,
	}

	if showPrev {
		p.Prev = fmt.Sprintf("/bingwallpapers/page/%d/%s?prev=1", wallpapers[0].Date, wallpapers[0].ID)
	}

	if err := view.WallpaperIndex.Execute(w, p); err != nil {
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
