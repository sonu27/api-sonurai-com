package store

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
