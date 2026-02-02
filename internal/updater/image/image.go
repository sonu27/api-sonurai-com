package image

import (
	"fmt"
	"strconv"
	"strings"

	"api/internal/updater/bing"
)

const bingURL = "https://www.bing.com"

// RGB represents a color as [R, G, B] values (0-255).
type RGB [3]int

// ToHex returns the color as a hex string (e.g., "#4A90D9").
func (c RGB) ToHex() string {
	return fmt.Sprintf("#%02X%02X%02X", c[0], c[1], c[2])
}

type Image struct {
	ID        string `json:"id" firestore:"id"`
	Title     string `json:"title,omitempty" firestore:"title,omitempty"`
	Copyright string `json:"copyright,omitempty" firestore:"copyright,omitempty"`
	Date      int    `json:"date,omitempty" firestore:"date,omitempty"`
	Market    string `json:"market,omitempty" firestore:"market,omitempty"`
	URLBase   string `json:"urlBase,omitempty" firestore:"urlBase,omitempty"`
	FullDesc  string `json:"fullDesc,omitempty" firestore:"fullDesc,omitempty"`

	Tags        map[string]float32 `json:"tags,omitempty" firestore:"tags,omitempty"`
	TagsOrdered []string           `json:"tagsOrdered,omitempty" firestore:"tagsOrdered,omitempty"`
	Colors      []string           `json:"colors,omitempty" firestore:"colors,omitempty"` // Hex strings like "#4A90D9"
}

func (i Image) URL() string {
	return i.URLBase + "_1920x1080.jpg"
}

func From(bw bing.Image, market string) (Image, error) {
	fullDesc := bw.Copyright
	id := strings.Replace(bw.URLBase, "/az/hprichbg/rb/", "", 1)
	id = strings.Replace(id, "/th?id=OHR.", "", 1)
	id = strings.Split(id, "_")[0]

	urlBase := bingURL + bw.URLBase

	date, err := strconv.Atoi(bw.StartDate)
	if err != nil {
		return Image{}, err
	}

	title, copyright, err := parseCopyright(bw.Copyright)
	if err != nil {
		return Image{}, err
	}

	image := Image{
		ID:        id,
		Title:     title,
		Copyright: copyright,
		Date:      date,
		Market:    market,
		URLBase:   urlBase,
		FullDesc:  fullDesc,
		Tags:      make(map[string]float32),
	}

	return image, nil
}

// parseCopyright extracts the title and copyright from Bing's copyright string.
// Bing formats: "Title (© Attribution)" or "Title（© Attribution）" (Chinese)
func parseCopyright(raw string) (title, copyright string, err error) {
	// Normalize: handle Chinese parentheses with space before © symbol
	// e.g., "【Title】 （ © Attribution ）" -> "【Title】 （© Attribution ）"
	normalized := strings.ReplaceAll(raw, "（ ©", "（©")

	// Try Chinese fullwidth parentheses first: （©
	if parts := strings.Split(normalized, "（©"); len(parts) == 2 {
		title = strings.TrimSpace(parts[0])
		copyright = strings.TrimSpace(strings.TrimSuffix(parts[1], "）"))
		return title, copyright, nil
	}

	// Try standard parentheses: (©
	if parts := strings.Split(raw, "(©"); len(parts) == 2 {
		title = strings.TrimSpace(parts[0])
		copyright = strings.TrimSpace(strings.TrimSuffix(parts[1], ")"))
		return title, copyright, nil
	}

	// Try just © symbol
	if parts := strings.Split(raw, "©"); len(parts) == 2 {
		title = strings.TrimSpace(parts[0])
		copyright = strings.TrimSpace(strings.TrimSuffix(parts[1], ")"))
		return title, copyright, nil
	}

	// No copyright symbol found - use the whole string as title
	if raw != "" {
		return strings.TrimSpace(raw), "", nil
	}

	return "", "", fmt.Errorf("unable to parse copyright from empty string")
}
