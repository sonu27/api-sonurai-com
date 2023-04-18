package view

import (
	"embed"
	"html/template"
)

//go:embed *
var templatesFS embed.FS

var (
	About          *template.Template
	WallpaperIndex *template.Template
	WallpaperGet   *template.Template
)

func init() {
	About = template.Must(template.ParseFS(templatesFS, "base.gohtml", "about.gohtml"))
	WallpaperIndex = template.Must(template.ParseFS(templatesFS, "base.gohtml", "wallpaper-index.gohtml"))
	WallpaperGet = template.Must(template.ParseFS(templatesFS, "base.gohtml", "wallpaper-get.gohtml"))
}
