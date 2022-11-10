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

var hc = http.Client{Timeout: 2 * time.Second}

func TestListWallpapers_ReturnAListOfWallpapers(t *testing.T) {
	w := []store.Wallpaper{{ID: "test1", Date: 20221221}}
	db := mocks.NewStorer(t)
	db.On("List", mock.Anything, mock.Anything).Return(w, nil)

	s := server.New("8080", db)
	ts := httptest.NewServer(s.Handler)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/wallpapers", nil)
	require.Nil(t, err)

	res, err := hc.Do(req)
	require.Nil(t, err)

	var listResponse server.ListResponse
	err = convertTo(res.Body, &listResponse)
	require.Nil(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, 1, len(listResponse.Data))
	assert.Equal(t, "test1", listResponse.Data[0].ID)
	assert.Empty(t, listResponse.Links.Prev)
	assert.Equal(t, "/wallpapers?startAfterDate=20221221&startAfterID=test1", listResponse.Links.Next)
}

func TestListWallpapers_ShowsPrev(t *testing.T) {
	w := []store.Wallpaper{{ID: "test1", Date: 20221027}}
	db := mocks.NewStorer(t)
	db.On("List", mock.Anything, mock.Anything).Return(w, nil)

	s := server.New("8080", db)
	ts := httptest.NewServer(s.Handler)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/wallpapers?startAfterDate=20221027&startAfterID=FrankensteinFriday", nil)
	require.Nil(t, err)

	res, err := hc.Do(req)
	require.Nil(t, err)

	var listResponse server.ListResponse
	err = convertTo(res.Body, &listResponse)
	require.Nil(t, err)

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, 1, len(listResponse.Data))
	assert.Equal(t, "test1", listResponse.Data[0].ID)
	assert.Equal(t, "/wallpapers?startAfterDate=20221027&startAfterID=test1&prev=1", listResponse.Links.Prev)
	assert.Equal(t, "/wallpapers?startAfterDate=20221027&startAfterID=test1", listResponse.Links.Next)
}

func convertTo(r io.ReadCloser, out any) error {
	b1, err := io.ReadAll(r)
	defer r.Close()

	m := make(map[string]any)
	if err := json.Unmarshal(b1, &m); err != nil {
		return err
	}

	b2, err := json.Marshal(m)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b2, out); err != nil {
		return err
	}

	return nil
}
