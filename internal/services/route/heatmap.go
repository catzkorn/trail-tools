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
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"
)

const (
	// heatmapPaddingMeters is how far around the waypoints of a leg the
	// user's activities are preferred, as routes can stray from the direct
	// line.
	heatmapPaddingMeters = 2000
	// heatmapSimplifyMeters is the tolerance activity tracks are simplified
	// with before routing prefers them.
	heatmapSimplifyMeters = 10
	// metersPerDegreeLat is the approximate length of a degree of latitude.
	metersPerDegreeLat = 111320.0
)

// heatmap is the user's activity tracks around the legs routed with the
// heatmap profile, simplified and clipped to the area.
type heatmap struct {
	tracks [][]geo.Point
}

// loadHeatmap loads the user's activity tracks around the points.
func (s *Service) loadHeatmap(ctx context.Context, userID pgtype.UUID, points []geo.Point) (*heatmap, error) {
	bounds, _ := geo.BoundsOf(points)
	bounds = padBounds(bounds, heatmapPaddingMeters)
	tracks, err := s.activities.ListTracksInBounds(ctx, userID, bounds, time.Time{}, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("failed to list activity tracks: %w", err)
	}
	heat := &heatmap{}
	for _, track := range tracks {
		points, err := TrackPoints(track)
		if err != nil {
			s.log.WarnContext(ctx, "Skipping undecodable track", slog.String("activity_id", store.UUIDToString(track.ActivityID)))
			continue
		}
		for _, part := range clip(points, bounds) {
			heat.tracks = append(heat.tracks, simplify(part, heatmapSimplifyMeters))
		}
	}
	return heat, nil
}

// near returns the tracks passing near a leg. It is nil without a heatmap.
func (h *heatmap) near(from, to geo.Point) [][]geo.Point {
	if h == nil {
		return nil
	}
	bounds, _ := geo.BoundsOf([]geo.Point{from, to})
	bounds = padBounds(bounds, heatmapPaddingMeters)
	var tracks [][]geo.Point
	for _, track := range h.tracks {
		if trackBounds, ok := geo.BoundsOf(track); ok && overlaps(trackBounds, bounds) {
			tracks = append(tracks, track)
		}
	}
	return tracks
}

func overlaps(a, b geo.Bounds) bool {
	return a.MinLat <= b.MaxLat && b.MinLat <= a.MaxLat && a.MinLng <= b.MaxLng && b.MinLng <= a.MaxLng
}

// clip returns the parts of a track inside bounds, each with at least two
// points.
func clip(points []geo.Point, b geo.Bounds) [][]geo.Point {
	var parts [][]geo.Point
	var part []geo.Point
	for _, p := range points {
		if p.Lat >= b.MinLat && p.Lat <= b.MaxLat && p.Lng >= b.MinLng && p.Lng <= b.MaxLng {
			part = append(part, p)
			continue
		}
		if len(part) >= 2 {
			parts = append(parts, part)
		}
		part = nil
	}
	if len(part) >= 2 {
		parts = append(parts, part)
	}
	return parts
}

// simplify removes points of a track that are within tolerance meters of the
// line through their neighbors (the Douglas-Peucker algorithm).
func simplify(points []geo.Point, tolerance float64) []geo.Point {
	if len(points) < 3 {
		return points
	}
	worst, worstIndex := 0.0, 0
	last := len(points) - 1
	for i := 1; i < last; i++ {
		if d := pointSegmentDistance(points[i], points[0], points[last]); d > worst {
			worst, worstIndex = d, i
		}
	}
	if worst <= tolerance {
		return []geo.Point{points[0], points[last]}
	}
	first := simplify(points[:worstIndex+1], tolerance)
	return append(first[:len(first)-1], simplify(points[worstIndex:], tolerance)...)
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

// pointSegmentDistance returns the distance in meters from p to the segment
// from a to b, in a flat projection around p, which is accurate over the
// short distances it is used for.
func pointSegmentDistance(p, a, b geo.Point) float64 {
	scale := math.Cos(p.Lat * math.Pi / 180)
	ax, ay := (a.Lng-p.Lng)*scale*metersPerDegreeLat, (a.Lat-p.Lat)*metersPerDegreeLat
	bx, by := (b.Lng-p.Lng)*scale*metersPerDegreeLat, (b.Lat-p.Lat)*metersPerDegreeLat
	dx, dy := bx-ax, by-ay
	f := 0.0
	if length := dx*dx + dy*dy; length > 0 {
		f = math.Max(0, math.Min(1, -(ax*dx+ay*dy)/length))
	}
	return math.Hypot(ax+f*dx, ay+f*dy)
}
