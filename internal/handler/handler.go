package handler

import (
	"context"
	"fmt"
	"strconv"

	"api/internal/api"
	"api/internal/store"
)

const (
	// DefaultPageSize is the default number of wallpapers returned per page.
	DefaultPageSize = 24

	// MaxTagScore is the maximum Vision API confidence score (used as initial cursor).
	MaxTagScore = 1.0
)

type Handler struct {
	store store.Storer
}

func New(store store.Storer) Handler {
	return Handler{store: store}
}

func (h Handler) GetRoot(_ context.Context) error {
	return nil
}

func (h Handler) GetWallpaper(ctx context.Context, p api.GetWallpaperParams) (api.GetWallpaperRes, error) {
	id := string(p.ID)

	var wp *store.WallpaperWithTags
	if i, err := strconv.Atoi(id); err == nil {
		wp, err = h.store.GetByOldID(ctx, i)
		if err != nil {
			return nil, err
		}
	} else {
		wp, err = h.store.Get(ctx, string(p.ID))
		if err != nil {
			return nil, err
		}
	}

	if wp == nil {
		return &api.GetWallpaperNotFound{}, nil
	}

	return &api.WallpaperWithTags{
		ID:        api.ID(wp.ID),
		Title:     wp.Title,
		Copyright: wp.Copyright,
		Date:      api.Date(wp.Date),
		Filename:  wp.Filename,
		Market:    wp.Market,
		Tags:      wp.Tags,
	}, nil
}

func (h Handler) GetWallpaperTags(ctx context.Context) (api.GetWallpaperTagsRes, error) {
	tags, err := h.store.GetTags(ctx)
	if err != nil {
		return nil, err
	}

	res := api.GetWallpaperTagsOK(tags)
	return &res, nil
}

func (h Handler) GetWallpapers(ctx context.Context, p api.GetWallpapersParams) (api.GetWallpapersRes, error) {
	q := store.ListQuery{Limit: DefaultPageSize}

	q.Reverse = p.Prev.Set

	if p.StartAfterDate.Set {
		q.StartAfterDate = int(p.StartAfterDate.Value)
	}

	if p.StartAfterID.Set {
		q.StartAfterID = string(p.StartAfterID.Value)
	}

	if p.Limit.Set {
		q.Limit = p.Limit.Value
	}

	wallpapers, err := h.store.List(ctx, q)
	if err != nil {
		return nil, err
	}

	if len(wallpapers) == 0 {
		return &api.GetWallpapersNotFound{}, nil
	}

	res := api.WallpaperList{
		Data: store.ToAPI(wallpapers),
	}

	last := wallpapers[len(wallpapers)-1]
	res.Links = api.Links{Next: api.NewOptString(fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s", last.Date, last.ID))}

	showPrev := p.StartAfterDate.Set && p.StartAfterID.Set
	if showPrev {
		first := wallpapers[0]
		res.Links.Prev = api.NewOptString(fmt.Sprintf("/wallpapers?startAfterDate=%d&startAfterID=%s&prev=1", first.Date, first.ID))
	}

	return &res, nil
}

func (h Handler) GetWallpapersByTag(ctx context.Context, p api.GetWallpapersByTagParams) (api.GetWallpapersByTagRes, error) {
	after := MaxTagScore
	if p.After.Set {
		after = p.After.Value
	}

	wallpapers, next, err := h.store.ListByTag(ctx, p.Tag, after)
	if err != nil {
		return nil, err
	}

	if len(wallpapers) == 0 {
		return &api.GetWallpapersByTagNotFound{}, nil
	}

	res := api.WallpaperList{
		Data: store.ToAPI(wallpapers),
	}

	if len(wallpapers) == store.TagPageSize && next > 0 {
		res.Links = api.Links{Next: api.NewOptString(fmt.Sprintf("/wallpapers/tags/%s?after=%.16f", p.Tag, next))}
	}

	return &res, nil
}
