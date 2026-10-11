package routing_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/routing"
)

// geofabrikIndex has a country with a smaller region inside it, and an
// extract without a PBF download.
const geofabrikIndex = `{"type": "FeatureCollection", "features": [
	{"properties": {"id": "switzerland", "name": "Switzerland", "urls": {"pbf": "https://example.com/switzerland.osm.pbf"}},
	 "geometry": {"type": "MultiPolygon", "coordinates": [[[[5.9, 45.8], [10.5, 45.8], [10.5, 47.8], [5.9, 47.8], [5.9, 45.8]]]]}},
	{"properties": {"id": "liechtenstein", "name": "Liechtenstein", "urls": {"pbf": "https://example.com/liechtenstein.osm.pbf"}},
	 "geometry": {"type": "Polygon", "coordinates": [[[9.47, 47.04], [9.64, 47.04], [9.64, 47.28], [9.47, 47.28], [9.47, 47.04]]]}},
	{"properties": {"id": "planet", "name": "Planet", "urls": {}},
	 "geometry": {"type": "Polygon", "coordinates": [[[-180, -90], [180, -90], [180, 90], [-180, 90], [-180, -90]]]}}
]}`

func TestExtractsFind(t *testing.T) {
	t.Parallel()
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(geofabrikIndex))
	}))
	t.Cleanup(srv.Close)
	extracts := routing.NewExtracts(srv.URL)
	ctx := context.Background()

	tests := []struct {
		name   string
		bounds geo.Bounds
		want   string
	}{
		{name: "smallest covering extract", bounds: geo.Bounds{MinLat: 47.1, MinLng: 9.5, MaxLat: 47.2, MaxLng: 9.6}, want: "Liechtenstein"},
		{name: "crossing a border", bounds: geo.Bounds{MinLat: 47.1, MinLng: 9.4, MaxLat: 47.2, MaxLng: 9.6}, want: "Switzerland"},
	}
	for _, tt := range tests {
		got, err := extracts.Find(ctx, tt.bounds)
		if err != nil || got.Name != tt.want {
			t.Errorf("%s: got %v, %v, want %s", tt.name, got, err, tt.want)
		}
	}
	if _, err := extracts.Find(ctx, geo.Bounds{MinLat: 50, MinLng: 9, MaxLat: 51, MaxLng: 10}); !errors.Is(err, routing.ErrNoExtract) {
		t.Errorf("got %v for an area no extract covers, want ErrNoExtract", err)
	}
	if requests != 1 {
		t.Errorf("the index was fetched %d times, want once", requests)
	}
}
