package strava

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PhotoSize is the size of the photos requested from Strava, in pixels on
// the long side.
const PhotoSize = 2048

// Photo is a photo added to an activity on Strava. The image is hosted on
// Strava's CDN.
type Photo struct {
	// Strava's ID of the photo.
	UniqueID string
	Caption  string
	// The URL of the image, at most PhotoSize pixels on the long side.
	URL string
	// The size of the image in pixels, zero if unknown.
	Width, Height int32
	// Where the photo was taken, if HasLocation.
	Lat, Lng    float64
	HasLocation bool
	// When the photo was taken, zero if unknown.
	CreatedAt time.Time

	// The raw JSON of the photo.
	Raw json.RawMessage
}

// photoResponse is a photo returned by Strava. The photos endpoint isn't in
// Strava's API reference, so any field may be missing.
type photoResponse struct {
	UniqueID  string            `json:"unique_id"`
	Caption   string            `json:"caption"`
	URLs      map[string]string `json:"urls"`
	Sizes     map[string][]int  `json:"sizes"`
	Location  []float64         `json:"location"`
	CreatedAt string            `json:"created_at"`
	VideoURL  string            `json:"video_url"`
}

// GetActivityPhotos gets the photos of an activity, in Strava's order.
// Videos, photos that are still processing and entries that can't be
// decoded are skipped.
func (c *Client) GetActivityPhotos(ctx context.Context, accessToken string, id int64) ([]*Photo, error) {
	q := url.Values{}
	q.Set("size", strconv.Itoa(PhotoSize))
	// Without photo_sources, only Instagram photos are returned.
	q.Set("photo_sources", "true")
	body, err := c.get(ctx, accessToken, "/activities/"+strconv.FormatInt(id, 10)+"/photos", q)
	if err != nil {
		return nil, err
	}
	var raws []json.RawMessage
	if err := json.Unmarshal(body, &raws); err != nil {
		return nil, fmt.Errorf("failed to decode photos: %w", err)
	}
	var photos []*Photo
	for _, raw := range raws {
		if p := decodePhoto(raw); p != nil {
			photos = append(photos, p)
		}
	}
	return photos, nil
}

// decodePhoto decodes a photo, or returns nil if it isn't a usable photo.
func decodePhoto(raw json.RawMessage) *Photo {
	var resp photoResponse
	if err := json.Unmarshal(raw, &resp); err != nil || resp.UniqueID == "" || resp.VideoURL != "" {
		return nil
	}
	key, imageURL := photoURL(resp.URLs)
	if imageURL == "" {
		return nil
	}
	p := &Photo{
		UniqueID: resp.UniqueID,
		Caption:  strings.TrimSpace(resp.Caption),
		URL:      imageURL,
		Raw:      raw,
	}
	if size := resp.Sizes[key]; len(size) == 2 && size[0] > 0 && size[1] > 0 {
		p.Width, p.Height = int32(size[0]), int32(size[1])
	}
	if loc := resp.Location; len(loc) == 2 {
		lat, lng := loc[0], loc[1]
		// Strava reports a missing location as 0, 0.
		valid := !math.IsNaN(lat) && !math.IsNaN(lng) && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
		if valid && (lat != 0 || lng != 0) {
			p.Lat, p.Lng, p.HasLocation = lat, lng, true
		}
	}
	if t, err := time.Parse(time.RFC3339, resp.CreatedAt); err == nil {
		p.CreatedAt = t
	}
	return p
}

// photoURL returns the URL of the requested size, or else of the largest
// size, with its key. Placeholders of photos that are still processing
// aren't returned.
func photoURL(urls map[string]string) (string, string) {
	usable := func(u string) bool {
		parsed, err := url.Parse(u)
		return err == nil && (parsed.Scheme == "https" || parsed.Scheme == "http") && parsed.Host != "" &&
			!strings.Contains(parsed.Path, "placeholder")
	}
	want := strconv.Itoa(PhotoSize)
	if u := urls[want]; usable(u) {
		return want, u
	}
	bestKey, bestSize := "", -1
	for key, u := range urls {
		size, err := strconv.Atoi(key)
		if err != nil || !usable(u) {
			continue
		}
		if size > bestSize || size == bestSize && key < bestKey {
			bestKey, bestSize = key, size
		}
	}
	if bestKey == "" {
		return "", ""
	}
	return bestKey, urls[bestKey]
}
