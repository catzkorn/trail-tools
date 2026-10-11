package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/catzkorn/trail-tools/internal/geo"
)

// DefaultGeofabrikIndexURL is the index of Geofabrik's OpenStreetMap
// extracts, with the area each covers.
const DefaultGeofabrikIndexURL = "https://download.geofabrik.de/index-v1.json"

// geofabrikIndexTTL is how long the index is cached. Geofabrik rarely adds
// extracts.
const geofabrikIndexTTL = 24 * time.Hour

// ErrNoExtract is returned when no extract covers an area.
var ErrNoExtract = errors.New("no OpenStreetMap extract covers the whole area, select a smaller area")

// Extract is an OpenStreetMap extract that can be imported.
type Extract struct {
	// The Geofabrik ID, e.g. "switzerland".
	ID string
	// The name, e.g. "Switzerland".
	Name string
	// The URL of the PBF file.
	URL string
	// The bounding box of the area the extract covers.
	Bounds geo.Bounds
}

// Extracts finds Geofabrik extracts covering an area. The index is fetched
// when first needed and cached.
type Extracts struct {
	url    string
	client *http.Client

	mu       sync.Mutex
	extracts []Extract
	fetched  time.Time
}

// NewExtracts creates a new Extracts using the index at url.
func NewExtracts(url string) *Extracts {
	return &Extracts{url: url, client: &http.Client{Timeout: 2 * time.Minute}}
}

// Find returns the smallest extract whose bounding box contains the bounds,
// as importing it reads the least data.
func (e *Extracts) Find(ctx context.Context, bounds geo.Bounds) (Extract, error) {
	extracts, err := e.index(ctx)
	if err != nil {
		return Extract{}, err
	}
	best, bestArea := Extract{}, math.Inf(1)
	for _, x := range extracts {
		b := x.Bounds
		if b.MinLat > bounds.MinLat || b.MaxLat < bounds.MaxLat || b.MinLng > bounds.MinLng || b.MaxLng < bounds.MaxLng {
			continue
		}
		if area := (b.MaxLat - b.MinLat) * (b.MaxLng - b.MinLng); area < bestArea {
			best, bestArea = x, area
		}
	}
	if best.URL == "" {
		return Extract{}, ErrNoExtract
	}
	return best, nil
}

func (e *Extracts) index(ctx context.Context) ([]Extract, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.extracts != nil && time.Since(e.fetched) < geofabrikIndexTTL {
		return e.extracts, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, e.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the extract index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch the extract index: status %d", resp.StatusCode)
	}
	extracts, err := parseGeofabrikIndex(resp.Body)
	if err != nil {
		return nil, err
	}
	e.extracts, e.fetched = extracts, time.Now()
	return extracts, nil
}

// parseGeofabrikIndex parses Geofabrik's GeoJSON index of extracts.
func parseGeofabrikIndex(r io.Reader) ([]Extract, error) {
	var index struct {
		Features []struct {
			Properties struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				URLs struct {
					PBF string `json:"pbf"`
				} `json:"urls"`
			} `json:"properties"`
			Geometry struct {
				Coordinates json.RawMessage `json:"coordinates"`
			} `json:"geometry"`
		} `json:"features"`
	}
	if err := json.NewDecoder(r).Decode(&index); err != nil {
		return nil, fmt.Errorf("failed to parse the extract index: %w", err)
	}
	var extracts []Extract
	for _, f := range index.Features {
		if f.Properties.URLs.PBF == "" {
			continue
		}
		var coordinates any
		if err := json.Unmarshal(f.Geometry.Coordinates, &coordinates); err != nil {
			continue
		}
		b := geo.Bounds{MinLat: 90, MinLng: 180, MaxLat: -90, MaxLng: -180}
		extendBounds(&b, coordinates)
		if b.MinLat > b.MaxLat {
			continue
		}
		extracts = append(extracts, Extract{
			ID:     f.Properties.ID,
			Name:   f.Properties.Name,
			URL:    f.Properties.URLs.PBF,
			Bounds: b,
		})
	}
	return extracts, nil
}

// extendBounds extends b with the positions in nested GeoJSON coordinate
// arrays, whatever the geometry type.
func extendBounds(b *geo.Bounds, coordinates any) {
	values, ok := coordinates.([]any)
	if !ok {
		return
	}
	if len(values) >= 2 {
		lng, lngOK := values[0].(float64)
		lat, latOK := values[1].(float64)
		if lngOK && latOK {
			b.MinLat, b.MaxLat = math.Min(b.MinLat, lat), math.Max(b.MaxLat, lat)
			b.MinLng, b.MaxLng = math.Min(b.MinLng, lng), math.Max(b.MaxLng, lng)
			return
		}
	}
	for _, v := range values {
		extendBounds(b, v)
	}
}
