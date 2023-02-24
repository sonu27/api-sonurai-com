package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api/internal/server"
	"api/internal/store"
	"api/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestIndexHandler(t *testing.T) {
	s := server.New("8080", nil)
	ts := httptest.NewServer(s.Handler)
	defer ts.Close()

	res, err := hc.Get(ts.URL)
	require.Nil(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)
}

func TestServer(t *testing.T) {
	tests := []struct {
		name     string
		req      request
		wantRes  any
		dbMockFn func(*mocks.Storer)
	}{
		{
			name: "ListWallpapers_ReturnsAListOfWallpapers",
			req: request{
				method: http.MethodGet,
				path:   "/wallpapers",
			},
			wantRes: server.ListResponse{
				Data: []store.Wallpaper{
					{ID: "test1", Date: 20221221},
					{ID: "test2", Date: 20221222},
				},
				Links: &server.Links{
					Next: "/wallpapers?startAfterDate=20221222&startAfterID=test2",
				},
			},
			dbMockFn: func(db *mocks.Storer) {
				w := []store.Wallpaper{
					{ID: "test1", Date: 20221221},
					{ID: "test2", Date: 20221222},
				}
				db.On("List", mock.Anything, mock.Anything).Return(w, nil)
			},
		},
		{
			name: "ListWallpapers_ShowsPrev",
			req: request{
				method: http.MethodGet,
				path:   "/wallpapers?startAfterDate=20221222&startAfterID=test2",
			},
			wantRes: server.ListResponse{
				Data: []store.Wallpaper{
					{ID: "test3", Date: 20221223},
					{ID: "test4", Date: 20221224},
				},
				Links: &server.Links{
					Prev: "/wallpapers?startAfterDate=20221223&startAfterID=test3&prev=1",
					Next: "/wallpapers?startAfterDate=20221224&startAfterID=test4",
				},
			},
			dbMockFn: func(db *mocks.Storer) {
				w := []store.Wallpaper{
					{ID: "test3", Date: 20221223},
					{ID: "test4", Date: 20221224},
				}
				db.On("List", mock.Anything, mock.Anything).Return(w, nil)
			},
		},
		{
			name: "ListWallpapersByTag_ReturnsAListOfWallpapers",
			req: request{
				method: http.MethodGet,
				path:   "/wallpapers/tags/test-tag",
			},
			wantRes: server.ListResponse{
				Data: []store.Wallpaper{
					{ID: "test1", Date: 20221221},
					{ID: "test2", Date: 20221222},
				},
			},
			dbMockFn: func(db *mocks.Storer) {
				w := []store.Wallpaper{
					{ID: "test1", Date: 20221221},
					{ID: "test2", Date: 20221222},
				}
				db.On("ListByTag", mock.Anything, mock.Anything, mock.Anything).Return(w, 0.999, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := mocks.NewStorer(t)
			s := server.New("8080", db)

			tt.dbMockFn(db)

			ts := httptest.NewServer(s.Handler)
			defer ts.Close()

			req, err := http.NewRequest(tt.req.method, ts.URL+tt.req.path, tt.req.body)
			require.Nil(t, err)

			res, err := hc.Do(req)
			require.Nil(t, err)

			response, err := convert[server.ListResponse](res.Body)
			require.Nil(t, err)

			assert.Equal(t, tt.wantRes, response)
		})
	}
}

func TestGetWallpaper_ReturnsAWallpaper(t *testing.T) {
	w := store.WallpaperWithTags{
		Wallpaper: store.Wallpaper{ID: "test1", Date: 20221221},
		Tags:      map[string]float32{"test": 0.999},
	}
	db := mocks.NewStorer(t)
	db.On("Get", mock.Anything, mock.Anything).Return(&w, nil)

	s := server.New("8080", db)
	ts := httptest.NewServer(s.Handler)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/wallpapers/test1", nil)
	require.Nil(t, err)

	res, err := hc.Do(req)
	require.Nil(t, err)

	getResponse, err := convert[store.WallpaperWithTags](res.Body)
	require.Nil(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, w, getResponse)
}

var hc = http.Client{Timeout: 2 * time.Second}

type request struct {
	method string
	path   string
	body   io.Reader
}

func convert[T any](r io.ReadCloser) (T, error) {
	var out T

	b1, err := io.ReadAll(r)
	defer r.Close()
	if err != nil {
		return out, err
	}

	m := make(map[string]any)
	if err := json.Unmarshal(b1, &m); err != nil {
		return out, err
	}

	b2, err := json.Marshal(m)
	if err != nil {
		return out, err
	}

	if err := json.Unmarshal(b2, &out); err != nil {
		return out, err
	}

	return out, err
}
