package model

type ListResponse struct {
	Data  []Wallpaper `json:"data"`
	Links *Links      `json:"links,omitempty"`
}

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

	Tags map[string]float64 `json:"tags"`
}

type Links struct {
	Prev string `json:"prev,omitempty"`
	Next string `json:"next,omitempty"`
}
