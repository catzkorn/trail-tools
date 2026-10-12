package mapdata_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/mapdata"
)

// fakeElevation returns elevations 1000 times the latitude.
type fakeElevation struct {
	lookups int
}

func (f *fakeElevation) Points(_ context.Context, points []geo.Point) ([]float64, error) {
	f.lookups += len(points)
	elevations := make([]float64, len(points))
	for i, p := range points {
		elevations[i] = p.Lat * 1000
	}
	return elevations, nil
}

func TestFillElevations(t *testing.T) {
	t.Parallel()
	elevation := &fakeElevation{}
	client := mapdata.New(mapdata.Config{Elevation: elevation})
	var points []geo.Point
	for i := range 250 {
		points = append(points, geo.Point{Lat: float64(i) * 0.001})
	}
	if err := client.FillElevations(context.Background(), points, 150); err != nil {
		t.Fatalf("FillElevations: %v", err)
	}
	if elevation.lookups > 150 {
		t.Errorf("looked up %d points, want at most 150", elevation.lookups)
	}
	for i, p := range points {
		if diff := p.Ele - p.Lat*1000; diff > 0.01 || diff < -0.01 {
			t.Fatalf("point %d: got elevation %v, want %v", i, p.Ele, p.Lat*1000)
		}
	}
}

func TestOverpassFallback(t *testing.T) {
	t.Parallel()
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "too busy", http.StatusGatewayTimeout)
	}))
	t.Cleanup(broken.Close)
	working := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if !strings.Contains(r.PostForm.Get("data"), `["natural"="peak"]`) {
			t.Errorf("unexpected query %q", r.PostForm.Get("data"))
		}
		fmt.Fprint(w, `{"elements":[
			{"type":"node","id":1,"lat":1.5,"lon":2.5,"tags":{"natural":"peak","name":"Top","ele":"1234"}},
			{"type":"way","id":2,"center":{"lat":3,"lon":4},"tags":{"amenity":"parking"}}
		]}`)
	}))
	t.Cleanup(working.Close)
	client := mapdata.New(mapdata.Config{OverpassURLs: []string{broken.URL, working.URL}})
	elements, err := client.FindFeatures(context.Background(), geo.Point{Lat: 1, Lng: 2}, 1000, []string{`["natural"="peak"]`})
	if err != nil {
		t.Fatalf("FindFeatures: %v", err)
	}
	if len(elements) != 2 {
		t.Fatalf("got %d elements", len(elements))
	}
	if loc, ok := elements[1].Location(); !ok || loc.Lat != 3 || loc.Lng != 4 {
		t.Errorf("unexpected way location %v", loc)
	}

	client = mapdata.New(mapdata.Config{OverpassURLs: []string{broken.URL}})
	if _, err := client.FindTrails(context.Background(), geo.Point{}, 1000); err == nil {
		t.Error("expected error when all instances fail")
	}
}

func TestStatusErrorPlainText(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		fmt.Fprint(w, `<?xml version="1.0"?><html><head><title>OSM3S Response</title></head>
<body><p>The data is from OpenStreetMap.</p>
<p><strong>Error</strong>: runtime error: the server is probably too busy.</p></body></html>`)
	}))
	t.Cleanup(srv.Close)
	client := mapdata.New(mapdata.Config{OverpassURLs: []string{srv.URL}})
	_, err := client.FindTrails(context.Background(), geo.Point{}, 1000)
	if err == nil || strings.Contains(err.Error(), "<") || !strings.Contains(err.Error(), "Error : runtime error: the server is probably too busy.") {
		t.Errorf("unexpected error %v", err)
	}
}
