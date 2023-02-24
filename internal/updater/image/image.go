package image

import (
	"api/internal/updater/bing_image"
	"strconv"
	"strings"
)

type Image struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Copyright string `json:"copyright"`
	Date      int    `json:"date"`
	Filename  string `json:"filename"`
	Market    string `json:"market"`
	FullDesc  string `json:"fullDesc"`
	URL       string `json:"url"`
	ThumbURL  string `json:"thumbUrl"`

	Tags map[string]float32 `json:"tags"`
}

func From(bw bing_image.Image, market string, bingURL string) (Image, error) {
	fullDesc := bw.Copyright
	id := strings.Replace(bw.URLBase, "/az/hprichbg/rb/", "", 1)
	id = strings.Replace(id, "/th?id=OHR.", "", 1)
	id = strings.Split(id, "_")[0]

	date, err := strconv.Atoi(bw.StartDate)
	if err != nil {
		return Image{}, err
	}

	var copyright string
	var title string

	if a := strings.Split(bw.Copyright, "（©"); len(a) == 2 {
		// chinese chars
		title = a[0]
		copyright = "© " + a[1]
		copyright = strings.Replace(copyright, "）", "", 1)
	} else if a := strings.Split(bw.Copyright, "(©"); len(a) == 2 {
		title = a[0]
		copyright = "© " + a[1]
		copyright = strings.Replace(copyright, ")", "", 1)
	} else {
		a := strings.Split(bw.Copyright, "©")
		title = a[0]
		copyright = "© " + a[1]
		copyright = strings.Replace(copyright, ")", "", 1)
	}

	title = strings.TrimSpace(title)
	copyright = strings.TrimSpace(copyright)

	image := Image{
		ID:        id,
		Title:     title,
		Copyright: copyright,
		Date:      date,
		Filename:  id,
		Market:    market,
		FullDesc:  fullDesc,
		URL:       bingURL + bw.URLBase + "_1920x1200.jpg",
		ThumbURL:  bingURL + bw.URLBase + "_1920x1080.jpg",
		Tags:      make(map[string]float32),
	}

	return image, nil
}
