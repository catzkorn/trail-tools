package route

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sync/atomic"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/routes"
	"github.com/catzkorn/trail-tools/internal/routing"
	"golang.org/x/sync/errgroup"
)

const (
	// snapToleranceMeters is how far a routed leg may stray from the recorded
	// track and still be considered to follow it. It allows for GPS error and
	// the offset between tracks and mapped ways.
	snapToleranceMeters = 40
	// snapStraightToleranceMeters is how far the track may stray from a
	// straight leg used where no mapped path follows it.
	snapStraightToleranceMeters = 25
	// snapSpacingMeters is the spacing the track is resampled to, which also
	// limits where waypoints can be placed.
	snapSpacingMeters = 20
	// minSnapLegMeters is the shortest section of track that is routed. A
	// shorter section that no route follows becomes straight legs.
	minSnapLegMeters = 150
	// maxSnapRoutingCalls limits the routing calls made to snap a track. The
	// rest of the track becomes straight legs once it is used up.
	maxSnapRoutingCalls = 400
	// maxSnapPathPoints limits the routed path points compared against the
	// track, to bound the cost of comparing long legs.
	maxSnapPathPoints = 500
	// maxShortestSnapMeters is the longest section routed with the shortest
	// profile when the trail profile doesn't follow the track. Over longer
	// sections, the route usually differs because of the waypoints rather
	// than the profile, so the call is saved.
	maxShortestSnapMeters = 3000
)

// snapProfiles are the profiles tried, in order, to follow the track.
var snapProfiles = []routes.Profile{routes.ProfileTrail, routes.ProfileShortest}

// snapLeg is a leg between two points of the resampled track, either routed
// or straight.
type snapLeg struct {
	from, to int
	profile  routes.Profile
	// The routed path, nil for straight legs.
	path *routing.Path
}

// trackSnapper fits a route with few waypoints to a recorded track.
type trackSnapper struct {
	s      *Service
	points []geo.Point
	dists  []float64
	// sem limits concurrent routing calls.
	sem   chan struct{}
	calls atomic.Int32
}

// snapTrack turns a recorded track into an editable route that follows it
// with as few waypoints as possible. Waypoints are added where routing
// between the existing ones strays from the track, starting with just the
// start and finish, and then removed again where routing around them still
// follows the track.
func (s *Service) snapTrack(ctx context.Context, track []geo.Point) ([]routes.LatLng, []routes.Leg, error) {
	points := resample(track, snapSpacingMeters)
	if len(points) < 2 {
		return nil, nil, fmt.Errorf("%w: the track is too short", errInvalidRoute)
	}
	t := &trackSnapper{
		s:      s,
		points: points,
		dists:  geo.CumulativeDistances(points),
		sem:    make(chan struct{}, maxConcurrentLegs),
	}
	legs, err := t.fit(ctx, 0, len(points)-1)
	if err != nil {
		return nil, nil, err
	}
	legs, err = t.merge(ctx, legs)
	if err != nil {
		return nil, nil, err
	}
	s.log.DebugContext(ctx, "Snapped track to the map", slog.Int("points", len(points)), slog.Int("legs", len(legs)), slog.Int("routing_calls", int(t.calls.Load())))
	if len(legs)+1 > maxWaypoints {
		return nil, nil, fmt.Errorf("%w: the track needs more than %d waypoints to follow; import it exactly instead", errInvalidRoute, maxWaypoints)
	}
	return t.build(ctx, legs)
}

// fit returns legs that follow the track from points[i] to points[j].
func (t *trackSnapper) fit(ctx context.Context, i, j int) ([]snapLeg, error) {
	split := -1
	if t.dists[j]-t.dists[i] >= minSnapLegMeters {
		leg, worst, err := t.route(ctx, i, j)
		if err != nil {
			return nil, err
		}
		if leg != nil {
			return []snapLeg{*leg}, nil
		}
		split = worst
	}
	if split < 0 {
		// The section is too short to route, or the routing budget is used
		// up: follow it with straight legs.
		return t.straight(i, j), nil
	}
	var first, second []snapLeg
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		first, err = t.fit(ctx, i, split)
		return err
	})
	g.Go(func() error {
		var err error
		second, err = t.fit(ctx, split, j)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return append(first, second...), nil
}

// route tries to route a leg from points[i] to points[j] that follows the
// track. If no profile does, it returns a nil leg and the index of the point
// to split the section at, or -1 if the routing budget is used up.
func (t *trackSnapper) route(ctx context.Context, i, j int) (*snapLeg, int, error) {
	worst := -1
	for _, profile := range snapProfiles {
		if profile == routes.ProfileShortest && t.dists[j]-t.dists[i] > maxShortestSnapMeters {
			continue
		}
		if t.calls.Add(1) > maxSnapRoutingCalls {
			return nil, -1, nil
		}
		path, err := t.routePath(ctx, t.points[i], t.points[j], profile)
		if err != nil {
			if ctx.Err() != nil {
				return nil, -1, ctx.Err()
			}
			if !errors.Is(err, routing.ErrNotRoutable) {
				return nil, -1, err
			}
			// The points may not be routable, e.g. far from any mapped way.
			continue
		}
		deviation, at := t.deviation(i, j, path.Points)
		if deviation <= snapToleranceMeters {
			return &snapLeg{from: i, to: j, profile: profile, path: path}, -1, nil
		}
		if worst < 0 {
			worst = at
		}
	}
	if worst <= i || worst >= j {
		// Split in the middle of the section.
		worst = i + (j-i)/2
		for k := i + 1; k < j; k++ {
			if t.dists[k]-t.dists[i] >= (t.dists[j]-t.dists[i])/2 {
				worst = k
				break
			}
		}
	}
	return nil, worst, nil
}

func (t *trackSnapper) routePath(ctx context.Context, from, to geo.Point, profile routes.Profile) (*routing.Path, error) {
	routingProfile := routing.ProfileTrail
	if profile == routes.ProfileShortest {
		routingProfile = routing.ProfileShortest
	}
	select {
	case t.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-t.sem }()
	return t.s.router.Route(ctx, routing.Request{From: from, To: to, Profile: routingProfile})
}

// deviation returns how far a path strays from the track between points[i]
// and points[j], in meters, and the index of the track point where it strays
// the most. Both the track's distance from the path and the path's distance
// from the track count, so that detours and shortcuts are both found.
func (t *trackSnapper) deviation(i, j int, path []geo.Point) (float64, int) {
	path = downsamplePoints(path, maxSnapPathPoints)
	section := t.points[i : j+1]
	worst, worstIndex := 0.0, i
	for k, p := range section {
		if d := distanceToPath(p, path); d > worst {
			worst, worstIndex = d, i+k
		}
	}
	for _, p := range path {
		if d := distanceToPath(p, section); d > worst {
			nearest, _ := geo.NearestIndex(section, p)
			worst, worstIndex = d, i+nearest
		}
	}
	return worst, worstIndex
}

// straight returns straight legs that follow the track from points[i] to
// points[j], with as few waypoints as possible.
func (t *trackSnapper) straight(i, j int) []snapLeg {
	worst, worstIndex := 0.0, -1
	for k := i + 1; k < j; k++ {
		if d := pointSegmentDistance(t.points[k], t.points[i], t.points[j]); d > worst {
			worst, worstIndex = d, k
		}
	}
	if worst <= snapStraightToleranceMeters {
		return []snapLeg{{from: i, to: j, profile: routes.ProfileStraight}}
	}
	return append(t.straight(i, worstIndex), t.straight(worstIndex, j)...)
}

// merge removes waypoints between two legs where a single leg still follows
// the track.
func (t *trackSnapper) merge(ctx context.Context, legs []snapLeg) ([]snapLeg, error) {
	for k := 0; k+1 < len(legs); {
		first, second := legs[k], legs[k+1]
		var merged *snapLeg
		switch {
		case first.path == nil && second.path == nil:
			if straight := t.straight(first.from, second.to); len(straight) == 1 {
				merged = &straight[0]
			}
		case first.path != nil && second.path != nil:
			leg, _, err := t.route(ctx, first.from, second.to)
			if err != nil {
				return nil, err
			}
			merged = leg
		}
		if merged == nil {
			k++
			continue
		}
		legs = append(append(legs[:k:k], *merged), legs[k+2:]...)
	}
	return legs, nil
}

// build creates the route's waypoints and legs. Waypoints next to a routed
// leg are moved to where routing snapped them onto the map.
func (t *trackSnapper) build(ctx context.Context, legs []snapLeg) ([]routes.LatLng, []routes.Leg, error) {
	waypoints := make([]routes.LatLng, len(legs)+1)
	for k := range waypoints {
		var p geo.Point
		switch {
		case k < len(legs) && legs[k].path != nil:
			p = legs[k].path.Points[0]
		case k > 0 && legs[k-1].path != nil:
			p = legs[k-1].path.Points[len(legs[k-1].path.Points)-1]
		case k < len(legs):
			p = t.points[legs[k].from]
		default:
			p = t.points[legs[k-1].to]
		}
		waypoints[k] = routes.LatLng{Lat: roundCoordinate(p.Lat), Lng: roundCoordinate(p.Lng)}
	}
	result := make([]routes.Leg, len(legs))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentLegs)
	for k, leg := range legs {
		from, to := waypoints[k], waypoints[k+1]
		if leg.path != nil {
			result[k] = legFromPath(from, to, leg.profile, leg.path)
			continue
		}
		g.Go(func() error {
			var err error
			result[k], err = t.s.routeLeg(ctx, nil, from, to, routes.ProfileStraight)
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, nil, err
	}
	return waypoints, result, nil
}

// roundCoordinate rounds a coordinate to 6 decimals, about 0.1 meters.
func roundCoordinate(v float64) float64 {
	return math.Round(v*1e6) / 1e6
}

// resample returns points of the track at least spacing meters apart, in a
// straight line, including its first and last points. Gaps longer than
// spacing, such as in low resolution polylines, are filled in, and GPS
// jitter between nearby points is dropped.
func resample(track []geo.Point, spacing float64) []geo.Point {
	if len(track) == 0 {
		return nil
	}
	result := []geo.Point{track[0]}
	for i := 1; i < len(track); i++ {
		last := result[len(result)-1]
		p := track[i]
		d := geo.Distance(last, p)
		if d < spacing {
			continue
		}
		if d >= 2*spacing {
			result = append(result, geo.Densify(last, p, spacing)[1:]...)
			continue
		}
		result = append(result, p)
	}
	if last, end := result[len(result)-1], track[len(track)-1]; last != end {
		result = append(result, end)
	}
	return result
}

// downsamplePoints returns at most limit points, evenly spread.
func downsamplePoints(points []geo.Point, limit int) []geo.Point {
	if len(points) <= limit {
		return points
	}
	idxs := geo.Downsample(len(points), limit)
	result := make([]geo.Point, len(idxs))
	for i, idx := range idxs {
		result[i] = points[idx]
	}
	return result
}

// distanceToPath returns the distance in meters from p to the nearest point
// on the path.
func distanceToPath(p geo.Point, path []geo.Point) float64 {
	if len(path) == 1 {
		return geo.Distance(p, path[0])
	}
	nearest := math.Inf(1)
	for i := 1; i < len(path); i++ {
		nearest = math.Min(nearest, pointSegmentDistance(p, path[i-1], path[i]))
	}
	return nearest
}

// trackCoverage returns the fraction of a track's length within the regions.
// Each segment counts as covered if its midpoint is.
func trackCoverage(track []geo.Point, regions []geo.Bounds) float64 {
	var total, covered float64
	for i := 1; i < len(track); i++ {
		d := geo.Distance(track[i-1], track[i])
		total += d
		mid := geo.Interpolate(track[i-1], track[i], 0.5)
		if slices.ContainsFunc(regions, func(b geo.Bounds) bool { return b.Contains(mid) }) {
			covered += d
		}
	}
	if total == 0 {
		return 0
	}
	return covered / total
}
