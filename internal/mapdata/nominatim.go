package mapdata

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// Place is a geocoding result.
type Place struct {
	Name     string
	Lat      float64
	Lng      float64
	Category string
	Type     string
	// The bounding box as south, north, west, east. Empty if unknown.
	BoundingBox []float64
}

type nominatimPlace struct {
	DisplayName string   `json:"display_name"`
	Lat         string   `json:"lat"`
	Lon         string   `json:"lon"`
	Category    string   `json:"category"`
	Type        string   `json:"type"`
	BoundingBox []string `json:"boundingbox"`
}

// Search geocodes a free text query using Nominatim.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Place, error) {
	if err := c.nominatim.wait(ctx); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("format", "jsonv2")
	q.Set("limit", strconv.Itoa(limit))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.conf.NominatimURL+"/search?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var resp []nominatimPlace
	if err := c.do(req, &resp); err != nil {
		return nil, fmt.Errorf("place search failed: %w", err)
	}
	places := make([]Place, 0, len(resp))
	for _, p := range resp {
		lat, err := strconv.ParseFloat(p.Lat, 64)
		if err != nil {
			continue
		}
		lng, err := strconv.ParseFloat(p.Lon, 64)
		if err != nil {
			continue
		}
		place := Place{Name: p.DisplayName, Lat: lat, Lng: lng, Category: p.Category, Type: p.Type}
		if len(p.BoundingBox) == 4 {
			for _, s := range p.BoundingBox {
				v, err := strconv.ParseFloat(s, 64)
				if err != nil {
					place.BoundingBox = nil
					break
				}
				place.BoundingBox = append(place.BoundingBox, v)
			}
		}
		places = append(places, place)
	}
	return places, nil
}
