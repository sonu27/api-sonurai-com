package server

import "api/internal/store"

type ListResponse struct {
	Data  []store.Wallpaper `json:"data"`
	Links *Links            `json:"links,omitempty"`
}

type Links struct {
	Prev string `json:"prev,omitempty"`
	Next string `json:"next,omitempty"`
}
