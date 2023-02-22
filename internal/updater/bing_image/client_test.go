package bing_image_test

import (
	"api/internal/updater/bing_image"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const bingResponse = `{
  "market":{
    "mkt":"en-US"
  },
  "images":[
    {
      "startdate":"20230220",
      "fullstartdate":"202302200800",
      "enddate":"20230221",
      "url":"/th?id=OHR.PresDayDC_EN-US2054662773_1920x1080.jpg&rf=LaDigue_1920x1080.jpg&pid=hp",
      "urlbase":"/th?id=OHR.PresDayDC_EN-US2054662773",
      "copyright":"Washington Monument and Capitol Building on the National Mall, Washington, DC (© AevanStock/Shutterstock)",
      "copyrightlink":"https://www.bing.com/search?q=presidents+day&form=hpcapt&filters=HpDate%3a%2220230220_0800%22",
      "title":"Happy Presidents Day!",
      "quiz":"/search?q=Bing+homepage+quiz&filters=WQOskey:%22HPQuiz_20230220_PresDayDC%22&FORM=HPQUIZ",
      "wp":true,
      "hsh":"5a45c7b3845a8d20ceccf5b78daa78db",
      "drk":1,
      "top":1,
      "bot":1,
      "hs":[]
    },
    {
      "startdate":"20230219",
      "fullstartdate":"202302190800",
      "enddate":"20230220",
      "url":"/th?id=OHR.MauiWhale_EN-US1928366389_1920x1080.jpg&rf=LaDigue_1920x1080.jpg&pid=hp",
      "urlbase":"/th?id=OHR.MauiWhale_EN-US1928366389",
      "copyright":"Humpback whales, Maui, Hawaii (© Flip Nicklin/Minden Pictures)",
      "copyrightlink":"https://www.bing.com/search?q=humpback+whale&form=hpcapt&filters=HpDate%3a%2220230219_0800%22",
      "title":"Migrating giants",
      "quiz":"/search?q=Bing+homepage+quiz&filters=WQOskey:%22HPQuiz_20230219_MauiWhale%22&FORM=HPQUIZ",
      "wp":true,
      "hsh":"75e73590a63a5d3bc6831f06d68b7c34",
      "drk":1,
      "top":1,
      "bot":1,
      "hs":[]
    }
  ],
  "tooltips":{
    "loading":"Loading...",
    "previous":"Previous image",
    "next":"Next image",
    "walle":"This image is not available to download as wallpaper.",
    "walls":"Download this image. Use of this image is restricted to wallpaper only."
  }
}`

func TestClient_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(bingResponse))
		require.Nil(t, err)
	}))
	defer server.Close()

	hc := &http.Client{Timeout: time.Second}
	c := bing_image.Client{BaseURL: server.URL, HC: hc}
	images, err := c.List(context.Background(), "en-US")
	assert.NoError(t, err)

	want := []bing_image.Image{
		{
			Copyright: "Washington Monument and Capitol Building on the National Mall, Washington, DC (© AevanStock/Shutterstock)",
			StartDate: "20230220",
			URL:       "/th?id=OHR.PresDayDC_EN-US2054662773_1920x1080.jpg&rf=LaDigue_1920x1080.jpg&pid=hp",
			URLBase:   "/th?id=OHR.PresDayDC_EN-US2054662773",
			WP:        true,
		},
		{
			Copyright: "Humpback whales, Maui, Hawaii (© Flip Nicklin/Minden Pictures)",
			StartDate: "20230219",
			URL:       "/th?id=OHR.MauiWhale_EN-US1928366389_1920x1080.jpg&rf=LaDigue_1920x1080.jpg&pid=hp",
			URLBase:   "/th?id=OHR.MauiWhale_EN-US1928366389",
			WP:        true,
		},
	}

	assert.Equal(t, images, want)
}
