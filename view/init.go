package view

import (
	"embed"
	"html/template"
)

//go:embed *
var templatesFS embed.FS

var (
	About              *template.Template
	Tags               *template.Template
	WallpaperIndex     *template.Template
	WallpaperGet       *template.Template
	WallpaperListByTag *template.Template
)

func init() {
	About = template.Must(template.ParseFS(templatesFS, "base.gohtml", "about.gohtml"))
	Tags = template.Must(template.ParseFS(templatesFS, "base.gohtml", "tags.gohtml"))
	WallpaperIndex = template.Must(template.ParseFS(templatesFS, "base.gohtml", "wallpaper-index.gohtml"))
	WallpaperGet = template.Must(template.ParseFS(templatesFS, "base.gohtml", "wallpaper-get.gohtml"))
	WallpaperListByTag = template.Must(template.ParseFS(templatesFS, "base.gohtml", "wallpaper-list-by-tag.gohtml"))
}
