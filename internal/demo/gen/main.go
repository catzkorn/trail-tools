// Command gen writes the demo data: the trails of Squamish from the Overpass
// API, and their elevations from the elevation tiles.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/catzkorn/trail-tools/internal/demo"
	"github.com/catzkorn/trail-tools/internal/elevation"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/mapdata"
	"github.com/catzkorn/trail-tools/internal/routing"
	"github.com/qedus/osmpbf"
)

func main() {
	ctx := context.Background()
	b := demo.Bounds
	ways, nodes, err := overpassWays(ctx, b)
	if err != nil {
		log.Fatal(err)
	}
	if err := write("squamish-trails.bin.gz", func(w io.Writer) error { return routing.WriteWays(w, ways, nodes) }); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Wrote %d ways with %d nodes\n", len(ways), len(nodes.IDs))
	patch, err := elevation.NewDEM(elevation.DefaultURL, elevation.DefaultCacheDir).Patch(ctx, b)
	if err != nil {
		log.Fatal(err)
	}
	if err := write("squamish-elevation.bin.gz", func(w io.Writer) error { _, err := patch.WriteTo(w); return err }); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Wrote the elevations")
}

// overpassWays queries the Overpass API for the ways within the bounds, and
// returns the foot accessible ones. The public servers are tried in turn, as
// they're often busy.
func overpassWays(ctx context.Context, b geo.Bounds) ([]routing.Way, *routing.Nodes, error) {
	query := fmt.Sprintf(`[out:json][timeout:180];way["highway"](%g,%g,%g,%g);(._;>;);out body;`, b.MinLat, b.MinLng, b.MaxLat, b.MaxLng)
	var errs []error
	for _, server := range mapdata.DefaultOverpassURLs {
		ways, nodes, err := queryOverpass(ctx, server, query, b)
		if err == nil {
			return ways, nodes, nil
		}
		log.Printf("%s: %v", server, err)
		errs = append(errs, err)
	}
	return nil, nil, errors.Join(errs...)
}

// queryOverpass runs a query for ways on an Overpass server, and returns
// the foot accessible ways within the bounds.
func queryOverpass(ctx context.Context, server, query string, b geo.Bounds) ([]routing.Way, *routing.Nodes, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, server, strings.NewReader(url.Values{"data": {query}}.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", mapdata.UserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var data struct {
		Elements []struct {
			Type  string            `json:"type"`
			ID    int64             `json:"id"`
			Lat   float64           `json:"lat"`
			Lon   float64           `json:"lon"`
			Nodes []int64           `json:"nodes"`
			Tags  map[string]string `json:"tags"`
		} `json:"elements"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, nil, err
	}
	nodes := map[int64]geo.Point{}
	var ways []*osmpbf.Way
	for _, e := range data.Elements {
		switch e.Type {
		case "node":
			nodes[e.ID] = geo.Point{Lat: e.Lat, Lng: e.Lon}
		case "way":
			ways = append(ways, &osmpbf.Way{ID: e.ID, Tags: e.Tags, NodeIDs: e.Nodes})
		}
	}
	footWays, footNodes := routing.WaysFromOSM(clip(ways, nodes, b), nodes)
	return footWays, footNodes, nil
}

// clip returns the pieces of the ways within the bounds, so that the
// elevations of the bounds cover all their nodes. Pieces of a way get IDs
// made from its ID.
func clip(ways []*osmpbf.Way, nodes map[int64]geo.Point, b geo.Bounds) []*osmpbf.Way {
	var result []*osmpbf.Way
	for _, w := range ways {
		var piece []int64
		pieces := 0
		flush := func() {
			if len(piece) >= 2 {
				result = append(result, &osmpbf.Way{ID: w.ID*16 + int64(pieces), Tags: w.Tags, NodeIDs: piece})
				pieces++
			}
			piece = nil
		}
		for _, id := range w.NodeIDs {
			if p, ok := nodes[id]; ok && b.Contains(p) {
				piece = append(piece, id)
			} else {
				flush()
			}
		}
		flush()
	}
	return result
}

func write(name string, fn func(io.Writer) error) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := fn(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
