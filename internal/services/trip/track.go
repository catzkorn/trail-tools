package trip

import (
	"fmt"
	"math"
	"time"

	activitiesv1 "github.com/johanbrandhorst/trail-tools/internal/gen/activities/v1"
	routesv1 "github.com/johanbrandhorst/trail-tools/internal/gen/routes/v1"
	"github.com/johanbrandhorst/trail-tools/internal/geo"
	"github.com/johanbrandhorst/trail-tools/internal/services/activity"
	"google.golang.org/protobuf/proto"
)

// maxTrackPoints is the number of points of the track copied into a report.
const maxTrackPoints = 2000

// trackTimeTolerance is how long before the start or after the end of the
// track a photo can be taken and still be placed at the start or end.
const trackTimeTolerance = 10 * time.Minute

// trackFromStreams returns the track to store in a report from an
// activity's streams: the time, distance, position and altitude,
// downsampled. It returns nil if the streams have no positions.
func trackFromStreams(streams *activitiesv1.ActivityStreams) *activitiesv1.ActivityStreams {
	n := len(streams.GetLat())
	if n < 2 || len(streams.GetLng()) != n {
		return nil
	}
	reduced := activitiesv1.ActivityStreams_builder{
		Lat: streams.GetLat(),
		Lng: streams.GetLng(),
	}.Build()
	if len(streams.GetTimeSeconds()) == n {
		reduced.SetTimeSeconds(streams.GetTimeSeconds())
	}
	if len(streams.GetDistanceMeters()) == n {
		reduced.SetDistanceMeters(streams.GetDistanceMeters())
	}
	if len(streams.GetAltitudeMeters()) == n {
		reduced.SetAltitudeMeters(streams.GetAltitudeMeters())
	}
	return activity.DownsampleStreams(reduced, maxTrackPoints)
}

// trackFromPolyline returns a track from an encoded polyline, for
// activities without streams. It returns nil if the polyline has less than
// two points.
func trackFromPolyline(polyline string) *activitiesv1.ActivityStreams {
	points, err := geo.DecodePolyline(polyline)
	if err != nil || len(points) < 2 {
		return nil
	}
	lats := make([]float64, len(points))
	lngs := make([]float64, len(points))
	for i, p := range points {
		lats[i], lngs[i] = p.Lat, p.Lng
	}
	return activitiesv1.ActivityStreams_builder{
		Lat:            lats,
		Lng:            lngs,
		DistanceMeters: geo.CumulativeDistances(points),
	}.Build()
}

// track is a decoded report track.
type track struct {
	points []geo.Point
	// The cumulative distance at each point in meters.
	dists []float64
	// The elapsed time at each point in seconds, nil if unknown.
	times []int32
}

// decodeTrack decodes a stored track. It returns nil for an empty track.
func decodeTrack(data []byte) (*track, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var streams activitiesv1.ActivityStreams
	if err := proto.Unmarshal(data, &streams); err != nil {
		return nil, fmt.Errorf("failed to decode track: %w", err)
	}
	return newTrack(&streams), nil
}

// newTrack returns the track of streams, or nil if it has less than two
// positions.
func newTrack(streams *activitiesv1.ActivityStreams) *track {
	n := len(streams.GetLat())
	if n < 2 || len(streams.GetLng()) != n {
		return nil
	}
	t := &track{points: make([]geo.Point, n)}
	hasAltitude := len(streams.GetAltitudeMeters()) == n
	for i := range n {
		t.points[i] = geo.Point{Lat: streams.GetLat()[i], Lng: streams.GetLng()[i], Ele: math.NaN()}
		if hasAltitude {
			t.points[i].Ele = streams.GetAltitudeMeters()[i]
		}
	}
	if len(streams.GetDistanceMeters()) == n {
		t.dists = streams.GetDistanceMeters()
	} else {
		t.dists = geo.CumulativeDistances(t.points)
	}
	if len(streams.GetTimeSeconds()) == n {
		t.times = streams.GetTimeSeconds()
	}
	return t
}

// positionAt returns the position on the track at an elapsed time in
// seconds, interpolated between the recorded points, and the distance along
// the track there. Times up to trackTimeTolerance before the start or after
// the end are placed at the start or end. It returns false if the track has
// no times, or the time is outside it.
func (t *track) positionAt(elapsed float64) (geo.Point, float64, bool) {
	if t == nil || len(t.times) == 0 {
		return geo.Point{}, 0, false
	}
	first, last := float64(t.times[0]), float64(t.times[len(t.times)-1])
	tolerance := trackTimeTolerance.Seconds()
	switch {
	case elapsed < first-tolerance || elapsed > last+tolerance:
		return geo.Point{}, 0, false
	case elapsed <= first:
		return t.points[0], t.dists[0], true
	case elapsed >= last:
		return t.points[len(t.points)-1], t.dists[len(t.dists)-1], true
	}
	// Find the first point at or after the time.
	lo, hi := 0, len(t.times)-1
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if float64(t.times[mid]) < elapsed {
			lo = mid
		} else {
			hi = mid
		}
	}
	span := float64(t.times[hi] - t.times[lo])
	f := 0.0
	if span > 0 {
		f = (elapsed - float64(t.times[lo])) / span
	}
	p := geo.Interpolate(t.points[lo], t.points[hi], f)
	return p, t.dists[lo] + f*(t.dists[hi]-t.dists[lo]), true
}

// nearestDistance returns the distance along the track of the point
// nearest to p.
func (t *track) nearestDistance(p geo.Point) (float64, bool) {
	if t == nil {
		return 0, false
	}
	i, _ := geo.NearestIndex(t.points, p)
	if i < 0 {
		return 0, false
	}
	return t.dists[i], true
}

// routePoints returns the track as route points. Unknown elevations are 0.
func (t *track) routePoints() []*routesv1.RoutePoint {
	if t == nil {
		return nil
	}
	points := make([]*routesv1.RoutePoint, len(t.points))
	for i, p := range t.points {
		ele := p.Ele
		if math.IsNaN(ele) {
			ele = 0
		}
		points[i] = routesv1.RoutePoint_builder{
			Lat:             p.Lat,
			Lng:             p.Lng,
			ElevationMeters: ele,
			DistanceMeters:  t.dists[i],
		}.Build()
	}
	return points
}

// visibleRange returns the indexes of the first and last point of the track
// outside the circles of the radius around its start and finish: from where
// it first leaves the circle around its start to where it last enters the
// circle around its finish. ok is false if no point is outside them.
func (t *track) visibleRange(radius float64) (from, to int, ok bool) {
	if t == nil || len(t.points) == 0 {
		return 0, 0, false
	}
	start, finish := t.points[0], t.points[len(t.points)-1]
	from, to = 0, len(t.points)-1
	for from <= to && geo.Distance(t.points[from], start) < radius {
		from++
	}
	for to >= from && geo.Distance(t.points[to], finish) < radius {
		to--
	}
	return from, to, from <= to
}

// nearEnds returns whether a point is within the radius of the start or
// finish of the track.
func (t *track) nearEnds(p geo.Point, radius float64) bool {
	if t == nil || len(t.points) == 0 || radius <= 0 {
		return false
	}
	return geo.Distance(p, t.points[0]) < radius || geo.Distance(p, t.points[len(t.points)-1]) < radius
}
