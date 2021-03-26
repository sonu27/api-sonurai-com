package model

type ListResponse struct {
	Data []ImageBasic `json:"data"`
}

type ImageBasic struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Copyright string `json:"copyright"`
	Date      int    `json:"date"`
	Filename  string `json:"filename"`
	Market    string `json:"market"`
}

type Image struct {
	ID        string             `json:"id"`
	Title     string             `json:"title"`
	Copyright string             `json:"copyright"`
	Date      int                `json:"date"`
	Filename  string             `json:"filename"`
	Market    string             `json:"market"`
	Tags      map[string]float64 `json:"tags"`
}
