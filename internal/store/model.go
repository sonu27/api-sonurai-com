package store

import (
	"fmt"

	"api/internal/api"
)

type Wallpaper struct {
	ID        string `json:"id" firestore:"id"`
	Title     string `json:"title" firestore:"title"`
	Copyright string `json:"copyright" firestore:"copyright"`
	Date      int    `json:"date" firestore:"date"`
	Filename  string `json:"filename" firestore:"filename"`
	Market    string `json:"market" firestore:"market"`
}

// RGB represents a color as [R, G, B] values (0-255).
type RGB [3]int

// ToHex returns the color as a hex string (e.g., "#4A90D9").
func (c RGB) ToHex() string {
	return fmt.Sprintf("#%02X%02X%02X", c[0], c[1], c[2])
}

type WallpaperWithTags struct {
	Wallpaper

	Tags   map[string]float32 `json:"tags" firestore:"tags"`
	Colors []RGB              `json:"colors,omitempty" firestore:"colors,omitempty"`
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
