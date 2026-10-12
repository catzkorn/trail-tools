package routing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/catzkorn/trail-tools/internal/elevation"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
)

// ImportConfig describes an OpenStreetMap import.
type ImportConfig struct {
	// The name of the region. Importing a region again replaces it.
	Region string
	// The OpenStreetMap PBF extracts: file paths or http(s) URLs, such as
	// Geofabrik extracts. Neighboring extracts can be imported together,
	// e.g. for an area across a border.
	Sources []string
	// Only ways with a node inside the bounds are imported. The whole extract
	// is imported if the bounds are zero.
	Bounds geo.Bounds
	// The elevation model.
	DEM *elevation.DEM
}

// Import loads the foot accessible ways of OpenStreetMap extracts into the
// routing graph, and returns the number of edges imported.
func Import(ctx context.Context, log *slog.Logger, db *store.DB, conf ImportConfig) (int, error) {
	start := time.Now()
	if len(conf.Sources) == 0 {
		return 0, errors.New("no OpenStreetMap extract to import")
	}
	var ways []Way
	nodes := &Nodes{}
	// Extracts are read one at a time, and only what's inside the bounds is
	// kept, so memory grows with the area rather than the extracts.
	for _, source := range conf.Sources {
		moreWays, moreNodes, err := readSource(ctx, log, source, conf.Bounds)
		if err != nil {
			return 0, err
		}
		log.InfoContext(ctx, "Read the OpenStreetMap extract", slog.String("source", source), slog.Int("ways", len(moreWays)), slog.Int("nodes", len(moreNodes.IDs)))
		ways, nodes = MergeExtracts(ways, nodes, moreWays, moreNodes)
	}
	elevations, err := conf.DEM.Elevations(ctx, nodes.Lats, nodes.Lngs)
	if err != nil {
		return 0, fmt.Errorf("failed to look up elevations: %w", err)
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
	count, err := SaveRegion(ctx, db, conf.Region, bounds, BuildEdges(ways, nodes, elevations, bounds))
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, fmt.Errorf("the extracts have no foot accessible ways within the bounds")
	}
	log.InfoContext(ctx, "Imported the region", slog.String("region", conf.Region), slog.Int("edges", count), slog.Duration("duration", time.Since(start)))
	return count, nil
}

// readSource reads an OpenStreetMap extract from a file or URL with ReadPBF.
// A downloaded extract is removed once read.
func readSource(ctx context.Context, log *slog.Logger, source string, bounds geo.Bounds) ([]Way, *Nodes, error) {
	path := source
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		log.InfoContext(ctx, "Downloading the OpenStreetMap extract", slog.String("url", source))
		downloaded, err := download(ctx, source)
		if err != nil {
			return nil, nil, err
		}
		defer os.Remove(downloaded)
		path = downloaded
	}
	return ReadPBF(ctx, path, bounds)
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
