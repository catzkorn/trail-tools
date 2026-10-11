package routing_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/routing"
	"github.com/catzkorn/trail-tools/internal/store/storetest"
)

// testGraph is a small network south to north from A (0, 0) to B (0.01, 0):
// a primary road straight between them, with a spur path from its middle,
// and two paths bowing east via C and west via D. A way far away is outside
// the bounds.
func testGraph() ([]routing.Way, *routing.Nodes, geo.Bounds) {
	nodes := &routing.Nodes{
		IDs:  []int64{1, 2, 3, 4, 5, 6, 7, 8},
		Lats: []float64{0, 0.01, 0.005, 0.005, 0.005, 0.005, 10, 10.01},
		Lngs: []float64{0, 0, 0.004, -0.0045, 0, 0.002, 10, 10},
	}
	ways := []routing.Way{
		{ID: 10, Nodes: []int64{1, 5, 2}, Highway: "primary", Terrain: geo.TerrainPaved},
		{ID: 11, Nodes: []int64{1, 3, 2}, Highway: "path", Terrain: geo.TerrainTrail},
		{ID: 12, Nodes: []int64{1, 4, 2}, Highway: "path", Terrain: geo.TerrainTrail},
		{ID: 13, Nodes: []int64{5, 6}, Highway: "path", Terrain: geo.TerrainTrail},
		{ID: 14, Nodes: []int64{7, 8}, Highway: "path", Terrain: geo.TerrainTrail},
	}
	return ways, nodes, geo.Bounds{MinLat: -0.01, MinLng: -0.01, MaxLat: 0.02, MaxLng: 0.02}
}

func TestBuildEdges(t *testing.T) {
	t.Parallel()
	ways, nodes, bounds := testGraph()
	elevations := make([]float64, len(nodes.IDs))
	edges := slices.Collect(routing.BuildEdges(ways, nodes, elevations, bounds))
	// The road is split at the spur, and the far way is left out.
	if len(edges) != 5 {
		t.Fatalf("got %d edges, want 5", len(edges))
	}
	if e := edges[0]; e.Highway != "primary" || e.Source != 1 || e.Target != 5 || len(e.Points) != 2 {
		t.Errorf("unexpected first edge %+v", e)
	}
	if e := edges[2]; e.Highway != "path" || e.Source != 1 || e.Target != 2 || len(e.Points) != 3 {
		t.Errorf("unexpected path edge %+v", e)
	}
}

func newRouter(t *testing.T) *routing.Router {
	t.Helper()
	db := storetest.New(t)
	ways, nodes, bounds := testGraph()
	elevations := make([]float64, len(nodes.IDs))
	edges := routing.BuildEdges(ways, nodes, elevations, bounds)
	ctx := context.Background()
	if _, err := routing.SaveRegion(ctx, db, "test", bounds, edges); err != nil {
		t.Fatalf("SaveRegion: %v", err)
	}
	// Importing again replaces the region.
	if count, err := routing.SaveRegion(ctx, db, "test", bounds, edges); err != nil || count != 5 {
		t.Fatalf("SaveRegion: %d edges, %v", count, err)
	}
	return routing.NewRouter(db)
}

// lngRange returns the westernmost and easternmost longitudes of a path.
func lngRange(path *routing.Path) (west, east float64) {
	west, east = math.Inf(1), math.Inf(-1)
	for _, p := range path.Points {
		west, east = math.Min(west, p.Lng), math.Max(east, p.Lng)
	}
	return west, east
}

func pathLength(path *routing.Path) float64 {
	dists := geo.CumulativeDistances(path.Points)
	return dists[len(dists)-1]
}

func TestRoute(t *testing.T) {
	t.Parallel()
	router := newRouter(t)
	ctx := context.Background()
	// Points on the road, near its ends.
	from, to := geo.Point{Lat: 0.0005}, geo.Point{Lat: 0.0095}

	// The shortest path takes the road.
	shortest, err := router.Route(ctx, routing.Request{From: from, To: to, Profile: routing.ProfileShortest})
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if west, east := lngRange(shortest); west < -0.0001 || east > 0.0001 {
		t.Errorf("shortest path strays from the road: %v to %v", west, east)
	}
	if got := pathLength(shortest); math.Abs(got-1002) > 2 {
		t.Errorf("got shortest length %v, want 1002", got)
	}
	if shortest.Terrains[0] != geo.TerrainPaved || shortest.Terrains[len(shortest.Terrains)-1] != geo.TerrainUnknown {
		t.Errorf("unexpected terrains %v", shortest.Terrains)
	}

	// The trail profile avoids the road and takes the shorter path, via C.
	trail, err := router.Route(ctx, routing.Request{From: from, To: to, Profile: routing.ProfileTrail})
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if _, east := lngRange(trail); math.Abs(east-0.004) > 0.0001 {
		t.Errorf("trail path reaches longitude %v, want 0.004", east)
	}
	if trail.Terrains[1] != geo.TerrainTrail {
		t.Errorf("unexpected terrains %v", trail.Terrains)
	}

	// Preferring a recording of the path via D takes it instead.
	recording := []geo.Point{{Lat: 0, Lng: 0.0001}, {Lat: 0.005, Lng: -0.0044}, {Lat: 0.01, Lng: 0.0001}}
	preferred, err := router.Route(ctx, routing.Request{From: from, To: to, Profile: routing.ProfileTrail, Prefer: [][]geo.Point{recording}})
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if west, _ := lngRange(preferred); math.Abs(west+0.0045) > 0.0001 {
		t.Errorf("preferred path reaches longitude %v, want -0.0045", west)
	}

	// Points on the same edge are routed along it.
	near, err := router.Route(ctx, routing.Request{From: geo.Point{Lat: 0.001}, To: geo.Point{Lat: 0.003}, Profile: routing.ProfileShortest})
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if got := pathLength(near); math.Abs(got-222) > 2 {
		t.Errorf("got length %v along one edge, want 222", got)
	}
}

func TestRegions(t *testing.T) {
	t.Parallel()
	router := newRouter(t)
	regions, err := router.Regions(context.Background())
	if err != nil {
		t.Fatalf("Regions: %v", err)
	}
	// Importing the region twice replaced it.
	if _, _, bounds := testGraph(); len(regions) != 1 || regions[0] != bounds {
		t.Errorf("got regions %v, want %v", regions, bounds)
	}
}

func TestRouteErrors(t *testing.T) {
	t.Parallel()
	router := newRouter(t)
	ctx := context.Background()
	start := geo.Point{Lat: 0.0002}
	for _, tc := range []struct {
		name string
		to   geo.Point
		want string
	}{
		{"far from a path", geo.Point{Lat: 0.015, Lng: 0.015}, "no mapped path within 500 meters"},
		{"outside the imported regions", geo.Point{Lat: 5, Lng: 5}, "no map data has been imported for the area"},
	} {
		_, err := router.Route(ctx, routing.Request{From: start, To: tc.to})
		if !errors.Is(err, routing.ErrNotRoutable) || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got error %v, want %q", tc.name, err, tc.want)
		}
	}
}

// writeTile writes a gzipped HGT tile with size samples on each side to dir.
func writeTile(t *testing.T, dir, name string, size int, sample func(row, col int) int16) {
	t.Helper()
	var raw bytes.Buffer
	for row := range size {
		for col := range size {
			if err := binary.Write(&raw, binary.BigEndian, sample(row, col)); err != nil {
				t.Fatal(err)
			}
		}
	}
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err := w.Write(raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), gz.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDEM(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A tile rising 100m per sample to the east, from 0 at the west edge,
	// with a void in the north east corner. Rows go south from the north.
	writeTile(t, dir, "N46E007.hgt.gz", 3, func(row, col int) int16 {
		if row == 0 && col == 2 {
			return -32768
		}
		return int16(col * 100)
	})
	// An empty file records a missing tile, e.g. over the sea.
	if err := os.WriteFile(filepath.Join(dir, "N00E000.hgt.gz"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// The URL is never used, as the tiles are cached.
	dem := routing.NewDEM("http://invalid.test", dir)
	elevations, err := dem.Elevations(context.Background(), []float64{46.5, 46.25, 0.5}, []float64{7.25, 7.75, 0.5})
	if err != nil {
		t.Fatalf("Elevations: %v", err)
	}
	if math.Abs(elevations[0]-50) > 0.01 {
		t.Errorf("got %v, want 50 a quarter of the way east", elevations[0])
	}
	if math.Abs(elevations[1]-150) > 0.01 {
		t.Errorf("got %v, want 150 three quarters of the way east", elevations[1])
	}
	if !math.IsNaN(elevations[2]) {
		t.Errorf("got %v for a missing tile, want NaN", elevations[2])
	}
}
