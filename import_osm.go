package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/routing"
	"github.com/catzkorn/trail-tools/internal/store"
	"gitlab.com/greyxor/slogor"
)

// importOSM runs the import-osm command, which loads an OpenStreetMap
// extract into the routing graph, and returns the exit code.
func importOSM(args []string) int {
	fs := flag.NewFlagSet("import-osm", flag.ExitOnError)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s import-osm [flags]\n\n", os.Args[0])
		fmt.Fprintln(fs.Output(), "Loads the foot accessible ways of an OpenStreetMap PBF extract into the routing graph.")
		fmt.Fprintln(fs.Output(), "Importing a region again replaces it. Extracts can be downloaded from https://download.geofabrik.de.")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	databaseURL := fs.String("database-url", "", "The URL of the postgres database")
	importID := fs.String("import-id", "", "Run an import started on the map data page, by its ID. Replaces -region, -pbf and -bounds")
	region := fs.String("region", "", "The name of the region, e.g. switzerland")
	source := fs.String("pbf", "", "The OpenStreetMap PBF extract: a file path or URL, e.g. https://download.geofabrik.de/europe/switzerland-latest.osm.pbf")
	bounds := fs.String("bounds", "", "Only import ways within south,west,north,east, e.g. 46.8,8.2,47.6,9.0. Defaults to the whole extract")
	demURL := fs.String("dem-url", routing.DefaultDEMURL, "The base URL of the HGT elevation tiles")
	demCache := fs.String("dem-cache", filepath.Join(os.TempDir(), "trail-tools-dem"), "The directory to cache elevation tiles in")
	logLevel := fs.String("log-level", "info", "The log level to use")
	logFormat := fs.String("log-format", "text", "The log format: \"text\" for terminals, or \"json\" for Cloud Logging")
	_ = fs.Parse(args)
	if err := setFlagsFromEnv(fs); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set flags from the environment: %v\n", err)
		return 1
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(*logLevel)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse log level: %v\n", err)
		return 1
	}
	var handler slog.Handler
	switch *logFormat {
	case "text":
		handler = slogor.NewHandler(os.Stdout, slogor.SetTimeFormat(time.Stamp), slogor.SetLevel(level))
	case "json":
		handler = newJSONLogHandler(level)
	default:
		fmt.Fprintf(os.Stderr, "Invalid log format %q, expected text or json\n", *logFormat)
		return 1
	}
	log := slog.New(handler)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := runImportOSM(ctx, log, *databaseURL, *importID, *region, *source, *bounds, *demURL, *demCache); err != nil {
		log.ErrorContext(ctx, "Failed to import the OpenStreetMap extract", slogor.Err(err))
		return 1
	}
	return 0
}

func runImportOSM(ctx context.Context, log *slog.Logger, databaseURL, importID, region, source, rawBounds, demURL, demCache string) error {
	switch {
	case databaseURL == "":
		return errors.New("database-url is required")
	case importID != "":
		// The import's region, extract and bounds are recorded.
	case region == "":
		return errors.New("region is required")
	case source == "":
		return errors.New("pbf is required")
	}
	bounds, err := parseBounds(rawBounds)
	if err != nil {
		return err
	}
	dbURL, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("failed to parse database-url: %w", err)
	}
	// Only log warnings from the database, as the import makes large queries.
	db, err := store.New(ctx, slog.New(slog.DiscardHandler), dbURL)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	defer db.Close()
	dem := routing.NewDEM(demURL, demCache)
	if importID != "" {
		id, err := store.StringToUUID(importID)
		if err != nil {
			return fmt.Errorf("invalid import-id: %w", err)
		}
		return routing.RunImport(ctx, log, db, dem, id)
	}
	_, err = routing.Import(ctx, log, db, routing.ImportConfig{
		Region: region,
		Source: source,
		Bounds: bounds,
		DEM:    dem,
	})
	return err
}

// parseBounds parses south,west,north,east bounds. Empty bounds are zero.
func parseBounds(s string) (geo.Bounds, error) {
	if s == "" {
		return geo.Bounds{}, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return geo.Bounds{}, fmt.Errorf("bounds must be south,west,north,east, got %q", s)
	}
	var values [4]float64
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return geo.Bounds{}, fmt.Errorf("invalid bounds %q: %w", s, err)
		}
		values[i] = v
	}
	b := geo.Bounds{MinLat: values[0], MinLng: values[1], MaxLat: values[2], MaxLng: values[3]}
	if b.MinLat >= b.MaxLat || b.MinLng >= b.MaxLng {
		return geo.Bounds{}, fmt.Errorf("bounds must have south < north and west < east, got %q", s)
	}
	return b, nil
}
