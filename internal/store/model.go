package store

import "api/internal/api"

type Wallpaper struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Copyright string `json:"copyright"`
	Date      int    `json:"date"`
	Filename  string `json:"filename"`
	Market    string `json:"market"`
}

type WallpaperWithTags struct {
	Wallpaper

	Tags map[string]float32 `json:"tags"`
}

func ToAPI(w []Wallpaper) []api.Wallpaper {
	res := make([]api.Wallpaper, len(w))
	for i, v := range w {
		res[i] = api.Wallpaper{
			ID:       api.ID(v.ID),
			Title:    v.Title,
			Date:     api.Date(v.Date),
			Filename: v.Filename,
			Market:   v.Market,
		}
	}
	return res
}
