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

// A trimmed BRouter response: a path of 4 points where the first stretch is a
// dirt path and the second an asphalt road.
const brouterResponse = `{
  "type": "FeatureCollection",
  "features": [{
    "type": "Feature",
    "properties": {
      "track-length": "333",
      "messages": [
        ["Longitude","Latitude","Elevation","Distance","CostPerKm","ElevCost","TurnCost","NodeCost","InitialCost","WayTags","NodeTags","Time","Energy"],
        ["0","1000","10","222","1000","0","0","0","0","highway=path surface=dirt","","0","0"],
        ["0","3000","30","111","1000","0","0","0","0","highway=tertiary surface=asphalt","","0","0"]
      ]
    },
    "geometry": {"type": "LineString", "coordinates": [[0,0,0],[0,0.001,10],[0,0.002,20],[0,0.003,30]]}
  }]
}`

func TestRoute(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("profile"); got != mapdata.ProfileHikingMountain {
			t.Errorf("unexpected profile %q", got)
		}
		if got := r.URL.Query().Get("lonlats"); got != "0.000000,0.000000|0.000000,0.003000" {
			t.Errorf("unexpected lonlats %q", got)
		}
		if r.Header.Get("User-Agent") != mapdata.UserAgent {
			t.Errorf("missing user agent")
		}
		fmt.Fprint(w, brouterResponse)
	}))
	t.Cleanup(srv.Close)
	client := mapdata.New(mapdata.Config{BRouterURL: srv.URL})
	path, err := client.Route(context.Background(), geo.Point{}, geo.Point{Lat: 0.003}, mapdata.ProfileHikingMountain)
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if len(path.Points) != 4 || path.Points[3].Ele != 30 {
		t.Fatalf("unexpected points %v", path.Points)
	}
	want := []geo.Terrain{geo.TerrainTrail, geo.TerrainTrail, geo.TerrainPaved, geo.TerrainUnknown}
	for i := range want {
		if path.Terrains[i] != want[i] {
			t.Errorf("terrain %d: got %q, want %q", i, path.Terrains[i], want[i])
		}
	}
}

func TestRouteError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "to-position not mapped in existing datafile", http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)
	client := mapdata.New(mapdata.Config{BRouterURL: srv.URL})
	_, err := client.Route(context.Background(), geo.Point{}, geo.Point{Lat: 1}, mapdata.ProfileShortest)
	if err == nil || !strings.Contains(err.Error(), "not mapped") {
		t.Errorf("unexpected error %v", err)
	}
}

func TestFillElevations(t *testing.T) {
	t.Parallel()
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		lats := strings.Split(r.URL.Query().Get("latitude"), ",")
		// Elevation is 1000 times the latitude.
		var elevations []string
		for _, lat := range lats {
			var v float64
			fmt.Sscan(lat, &v)
			elevations = append(elevations, fmt.Sprint(v*1000))
		}
		fmt.Fprintf(w, `{"elevation":[%s]}`, strings.Join(elevations, ","))
	}))
	t.Cleanup(srv.Close)
	client := mapdata.New(mapdata.Config{ElevationURL: srv.URL})
	var points []geo.Point
	for i := range 250 {
		points = append(points, geo.Point{Lat: float64(i) * 0.001})
	}
	if err := client.FillElevations(context.Background(), points, 150); err != nil {
		t.Fatalf("FillElevations: %v", err)
	}
	if requests != 2 {
		t.Errorf("got %d requests, want 2", requests)
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
