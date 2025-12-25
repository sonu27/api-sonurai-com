package store

import "api/internal/api"

type Wallpaper struct {
	ID        string `json:"id" firestore:"id"`
	Title     string `json:"title" firestore:"title"`
	Copyright string `json:"copyright" firestore:"copyright"`
	Date      int    `json:"date" firestore:"date"`
	Filename  string `json:"filename" firestore:"filename"`
	Market    string `json:"market" firestore:"market"`
}

type WallpaperWithTags struct {
	Wallpaper

	Tags map[string]float32 `json:"tags" firestore:"tags"`
}

func ToAPI(w []Wallpaper) []api.Wallpaper {
	res := make([]api.Wallpaper, len(w))
	for i, v := range w {
		res[i] = api.Wallpaper{
			ID:        api.ID(v.ID),
			Title:     v.Title,
			Copyright: v.Copyright,
			Date:      api.Date(v.Date),
			Filename:  v.Filename,
			Market:    v.Market,
		}
	}
	return res
}
