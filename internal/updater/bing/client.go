package bing

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type ListResponse struct {
	Market struct {
		Market string `json:"mkt"`
	} `json:"market"`
	Images []Image `json:"images"`
}

type Image struct {
	Copyright string `json:"copyright"`
	StartDate string `json:"startdate"`
	URL       string `json:"url"`
	URLBase   string `json:"urlbase"`
	WP        bool   `json:"wp"`
}

type Client struct {
	BaseURL string
	HC      *http.Client
}

func (c *Client) List(_ context.Context, market string) ([]Image, error) {
	resp, err := c.HC.Get(c.BaseURL + "/HPImageArchive.aspx?format=js&n=8&mbl=1&mkt=" + market)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	lr := new(ListResponse)
	if err := json.NewDecoder(resp.Body).Decode(lr); err != nil {
		return nil, err
	}

	if market != lr.Market.Market {
		log.Printf("warning: market mismatch: requested %s, got %s", market, lr.Market.Market)
	}

	return lr.Images, nil
}
