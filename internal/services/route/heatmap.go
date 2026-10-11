package route

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/catzkorn/trail-tools/internal/activities"
	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/mapdata"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

const (
	// heatmapCellMeters is the size of the grid cells that routed paths are
	// matched against the user's activities in. A path is on the heatmap if
	// an activity passed through its cell or a neighboring cell, which
	// tolerates GPS error and the offset between tracks and mapped ways.
	heatmapCellMeters = 30
	// heatmapPaddingMeters is how far around the waypoints of a leg to look
	// for activities, as alternative routes can stray from the direct line.
	heatmapPaddingMeters = 2000
	// maxHeatmapDetour is the longest an alternative route may be, relative to
	// the best trail route, to be picked for running on the heatmap.
	maxHeatmapDetour = 1.5
	// metersPerDegreeLat is the approximate length of a degree of latitude.
	metersPerDegreeLat = 111320.0
)

// heatmap is the set of grid cells that the user's activities passed
// through.
type heatmap struct {
	grid  *geo.Grid
	cells map[int]bool
}

// loadHeatmap loads the user's heatmap around the points.
func (s *Service) loadHeatmap(ctx context.Context, userID pgtype.UUID, points []geo.Point) (*heatmap, error) {
	bounds, _ := geo.BoundsOf(points)
	bounds = padBounds(bounds, heatmapPaddingMeters)
	tracks, err := s.activities.ListTracksInBounds(ctx, userID, bounds, time.Time{}, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("failed to list activity tracks: %w", err)
	}
	heat := &heatmap{grid: geo.NewGrid(bounds, heatmapCellMeters), cells: map[int]bool{}}
	for _, track := range tracks {
		points, err := TrackPoints(track)
		if err != nil {
			s.log.WarnContext(ctx, "Skipping undecodable track", slog.String("activity_id", store.UUIDToString(track.ActivityID)))
			continue
		}
		for cell := range heat.grid.Cells(points) {
			heat.cells[cell] = true
		}
	}
	return heat, nil
}

// padBounds grows bounds by about padding meters on each side.
func padBounds(b geo.Bounds, padding float64) geo.Bounds {
	midLat := (b.MinLat + b.MaxLat) / 2
	padLat := padding / metersPerDegreeLat
	padLng := padding / (metersPerDegreeLat * math.Max(0.01, math.Cos(midLat*math.Pi/180)))
	return geo.Bounds{
		MinLat: math.Max(-90, b.MinLat-padLat),
		MinLng: math.Max(-180, b.MinLng-padLng),
		MaxLat: math.Min(90, b.MaxLat+padLat),
		MaxLng: math.Min(180, b.MaxLng+padLng),
	}
}

// near reports whether an activity passed through the cell containing p or
// one of its neighbors.
func (h *heatmap) near(p geo.Point) bool {
	cell, ok := h.grid.Cell(p)
	if !ok {
		return false
	}
	row, col := cell/h.grid.Cols, cell%h.grid.Cols
	for r := row - 1; r <= row+1; r++ {
		for c := col - 1; c <= col+1; c++ {
			if r < 0 || r >= h.grid.Rows || c < 0 || c >= h.grid.Cols {
				continue
			}
			if h.cells[r*h.grid.Cols+c] {
				return true
			}
		}
	}
	return false
}

// anyWithin reports whether an activity passed through the bounds.
func (h *heatmap) anyWithin(b geo.Bounds) bool {
	for cell := range h.cells {
		lat, lng := h.grid.Center(cell)
		if lat >= b.MinLat && lat <= b.MaxLat && lng >= b.MinLng && lng <= b.MaxLng {
			return true
		}
	}
	return false
}

// overlap returns the length of the path in meters and the fraction of it
// that is on the heatmap.
func (h *heatmap) overlap(points []geo.Point) (length, fraction float64) {
	var covered float64
	for i := 1; i < len(points); i++ {
		section := geo.Densify(points[i-1], points[i], heatmapCellMeters/2)
		for j := 1; j < len(section); j++ {
			d := geo.Distance(section[j-1], section[j])
			length += d
			if h.near(geo.Interpolate(section[j-1], section[j], 0.5)) {
				covered += d
			}
		}
	}
	if length == 0 {
		return 0, 0
	}
	return length, covered / length
}

// routeOnHeatmap routes a trail leg that prefers the paths the user has run
// before: of BRouter's alternative routes, it picks the one with the largest
// share on the user's heatmap. Alternatives much longer than the best route
// are ignored. Without activities near the leg, it is the best trail route.
func (s *Service) routeOnHeatmap(ctx context.Context, heat *heatmap, from, to geo.Point) (*mapdata.RoutedPath, error) {
	best, err := s.mapData.Route(ctx, from, to, mapdata.ProfileHikingMountain, 0)
	if err != nil {
		return nil, err
	}
	if heat == nil {
		return best, nil
	}
	bounds, _ := geo.BoundsOf([]geo.Point{from, to})
	if !heat.anyWithin(padBounds(bounds, heatmapPaddingMeters/2)) {
		return best, nil
	}
	bestLength, bestFraction := heat.overlap(best.Points)
	if bestFraction >= 1 {
		return best, nil
	}
	alternatives := make([]*mapdata.RoutedPath, mapdata.MaxAlternatives)
	g, ctx := errgroup.WithContext(ctx)
	for i := 1; i < mapdata.MaxAlternatives; i++ {
		g.Go(func() error {
			// Not every leg has alternatives, so failures are ignored.
			path, err := s.mapData.Route(ctx, from, to, mapdata.ProfileHikingMountain, i)
			if err == nil {
				alternatives[i] = path
			}
			return nil
		})
	}
	_ = g.Wait()
	maxLength := bestLength * maxHeatmapDetour
	for _, path := range alternatives {
		if path == nil {
			continue
		}
		length, fraction := heat.overlap(path.Points)
		if length <= maxLength && fraction > bestFraction {
			best, bestFraction = path, fraction
		}
	}
	return best, nil
}

// TrackPoints returns the points of an activity's GPS track, from its
// streams if they have been synced or its summary polyline otherwise.
// Elevations are not set.
func TrackPoints(track activities.Track) ([]geo.Point, error) {
	if track.Streams != nil {
		var streams activitiesv1.ActivityStreams
		if err := proto.Unmarshal(track.Streams, &streams); err != nil {
			return nil, err
		}
		lats, lngs := streams.GetLat(), streams.GetLng()
		points := make([]geo.Point, min(len(lats), len(lngs)))
		for i := range points {
			points[i] = geo.Point{Lat: lats[i], Lng: lngs[i]}
		}
		if len(points) > 0 {
			return points, nil
		}
	}
	return geo.DecodePolyline(track.SummaryPolyline)
}
