package routing

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
)

// ImportConfig describes an OpenStreetMap import.
type ImportConfig struct {
	// The name of the region. Importing a region again replaces it.
	Region string
	// The OpenStreetMap PBF extract: a file path or an http(s) URL, such as
	// a Geofabrik extract.
	Source string
	// Only ways with a node inside the bounds are imported. The whole extract
	// is imported if the bounds are zero.
	Bounds geo.Bounds
	// The elevation model.
	DEM *DEM
}

// Import loads the foot accessible ways of an OpenStreetMap extract into the
// routing graph.
func Import(ctx context.Context, log *slog.Logger, db *store.DB, conf ImportConfig) error {
	start := time.Now()
	path := conf.Source
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		log.InfoContext(ctx, "Downloading the OpenStreetMap extract", slog.String("url", conf.Source))
		downloaded, err := download(ctx, conf.Source)
		if err != nil {
			return err
		}
		defer os.Remove(downloaded)
		path = downloaded
	}
	ways, nodes, err := ReadPBF(ctx, func() (io.ReadCloser, error) { return os.Open(path) })
	if err != nil {
		return err
	}
	log.InfoContext(ctx, "Read the OpenStreetMap extract", slog.Int("ways", len(ways)), slog.Int("nodes", len(nodes.IDs)))
	elevations, err := conf.DEM.Elevations(ctx, nodes.Lats, nodes.Lngs)
	if err != nil {
		return fmt.Errorf("failed to look up elevations: %w", err)
	}
	bounds := conf.Bounds
	if bounds == (geo.Bounds{}) {
		bounds = geo.Bounds{MinLat: 90, MinLng: 180, MaxLat: -90, MaxLng: -180}
		for i := range nodes.IDs {
			if !math.IsNaN(nodes.Lats[i]) {
				bounds.MinLat, bounds.MaxLat = math.Min(bounds.MinLat, nodes.Lats[i]), math.Max(bounds.MaxLat, nodes.Lats[i])
				bounds.MinLng, bounds.MaxLng = math.Min(bounds.MinLng, nodes.Lngs[i]), math.Max(bounds.MaxLng, nodes.Lngs[i])
			}
		}
	}
	edges := BuildEdges(ways, nodes, elevations, bounds)
	if len(edges) == 0 {
		return fmt.Errorf("the extract has no foot accessible ways within the bounds")
	}
	if err := SaveRegion(ctx, db, conf.Region, bounds, edges); err != nil {
		return err
	}
	log.InfoContext(ctx, "Imported the region", slog.String("region", conf.Region), slog.Int("edges", len(edges)), slog.Duration("duration", time.Since(start)))
	return nil
}

// download saves a URL to a temporary file and returns its path.
func download(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download %s: status %d", url, resp.StatusCode)
	}
	f, err := os.CreateTemp("", "osm-*.pbf")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to download %s: %w", url, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
