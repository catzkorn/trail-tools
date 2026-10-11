package route

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/mapdata"
	"github.com/catzkorn/trail-tools/internal/routes"
	"golang.org/x/sync/errgroup"
)

const (
	// maxWaypoints is the maximum number of waypoints in a route.
	maxWaypoints = 200
	// straightSpacingMeters is the spacing of points on straight legs, used
	// to look up elevations.
	straightSpacingMeters = 25
	// maxElevationLookups is the maximum number of elevation lookups for a
	// single track.
	maxElevationLookups = 500
	// importWaypointSpacingMeters is the distance between waypoints created
	// when importing a track, so it can be edited.
	importWaypointSpacingMeters = 1000
	// maxImportWaypoints is the maximum number of waypoints created when
	// importing a track.
	maxImportWaypoints = 60
	// maxConcurrentLegs is the maximum number of legs routed concurrently.
	maxConcurrentLegs = 4
)

// MapData provides routing and elevation data.
type MapData interface {
	Route(ctx context.Context, from, to geo.Point, profile string) (*mapdata.RoutedPath, error)
	FillElevations(ctx context.Context, points []geo.Point, maxLookups int) error
}

// errInvalidRoute is wrapped by errors caused by invalid input, such as
// waypoints that can't be routed.
var errInvalidRoute = errors.New("invalid route")

// buildLegs computes the legs between consecutive waypoints. Existing legs
// with the same endpoints are reused, unless they were routed with a
// different profile.
func buildLegs(ctx context.Context, md MapData, waypoints []routes.LatLng, profile routes.Profile, existing []routes.Leg) ([]routes.Leg, error) {
	if len(waypoints) < 2 {
		return nil, fmt.Errorf("%w: at least 2 waypoints are required", errInvalidRoute)
	}
	if len(waypoints) > maxWaypoints {
		return nil, fmt.Errorf("%w: at most %d waypoints are allowed", errInvalidRoute, maxWaypoints)
	}
	for i, wp := range waypoints {
		if math.IsNaN(wp.Lat) || math.IsNaN(wp.Lng) || wp.Lat < -90 || wp.Lat > 90 || wp.Lng < -180 || wp.Lng > 180 {
			return nil, fmt.Errorf("%w: waypoint %d has invalid coordinates", errInvalidRoute, i+1)
		}
	}
	legs := make([]routes.Leg, len(waypoints)-1)
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrentLegs)
	for i := range legs {
		from, to := waypoints[i], waypoints[i+1]
		if leg, ok := findLeg(existing, from, to, profile); ok {
			legs[i] = leg
			continue
		}
		g.Go(func() error {
			leg, err := routeLeg(ctx, md, from, to, profile)
			if err != nil {
				return fmt.Errorf("could not compute the leg from waypoint %d to %d: %w", i+1, i+2, err)
			}
			legs[i] = leg
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return legs, nil
}

func findLeg(existing []routes.Leg, from, to routes.LatLng, profile routes.Profile) (routes.Leg, bool) {
	for _, leg := range existing {
		// Imported legs have no profile and are kept until their endpoints
		// move.
		if leg.From == from && leg.To == to && (leg.Profile == "" || leg.Profile == profile) {
			return leg, true
		}
	}
	return routes.Leg{}, false
}

func routeLeg(ctx context.Context, md MapData, from, to routes.LatLng, profile routes.Profile) (routes.Leg, error) {
	fromPoint := geo.Point{Lat: from.Lat, Lng: from.Lng}
	toPoint := geo.Point{Lat: to.Lat, Lng: to.Lng}
	switch profile {
	case routes.ProfileStraight:
		points := geo.Densify(fromPoint, toPoint, straightSpacingMeters)
		if err := md.FillElevations(ctx, points, maxElevationLookups); err != nil {
			return routes.Leg{}, err
		}
		return routes.NewLeg(from, to, profile, points, nil), nil
	case routes.ProfileTrail, routes.ProfileShortest:
		brouterProfile := mapdata.ProfileHikingMountain
		if profile == routes.ProfileShortest {
			brouterProfile = mapdata.ProfileShortest
		}
		path, err := md.Route(ctx, fromPoint, toPoint, brouterProfile)
		if err != nil {
			return routes.Leg{}, fmt.Errorf("%w: %w; try moving the waypoints closer to a trail or road, or use the STRAIGHT profile", errInvalidRoute, err)
		}
		// Make the leg start and end exactly at the waypoints, so the route
		// is continuous even if routing snapped them to the nearest way.
		points := append([]geo.Point{{Lat: from.Lat, Lng: from.Lng, Ele: path.Points[0].Ele}}, path.Points...)
		points = append(points, geo.Point{Lat: to.Lat, Lng: to.Lng, Ele: path.Points[len(path.Points)-1].Ele})
		terrains := append([]geo.Terrain{geo.TerrainUnknown}, path.Terrains...)
		terrains[len(terrains)-1] = geo.TerrainUnknown
		terrains = append(terrains, geo.TerrainUnknown)
		points, terrains = dedupe(points, terrains)
		return routes.NewLeg(from, to, profile, points, terrains), nil
	default:
		return routes.Leg{}, fmt.Errorf("%w: unknown profile %q", errInvalidRoute, profile)
	}
}

// dedupe removes consecutive duplicate points, keeping the terrain of the
// segment that follows.
func dedupe(points []geo.Point, terrains []geo.Terrain) ([]geo.Point, []geo.Terrain) {
	var outPoints []geo.Point
	var outTerrains []geo.Terrain
	for i, p := range points {
		if n := len(outPoints); n > 0 && outPoints[n-1].Lat == p.Lat && outPoints[n-1].Lng == p.Lng {
			outTerrains[n-1] = terrains[i]
			continue
		}
		outPoints = append(outPoints, p)
		outTerrains = append(outTerrains, terrains[i])
	}
	return outPoints, outTerrains
}

// legsFromTrack splits a recorded or imported track into editable legs, with
// waypoints spaced along the track.
func legsFromTrack(points []geo.Point) ([]routes.LatLng, []routes.Leg) {
	dists := geo.CumulativeDistances(points)
	total := dists[len(dists)-1]
	spacing := math.Max(importWaypointSpacingMeters, total/maxImportWaypoints)
	idxs := []int{0}
	next := spacing
	for i := 1; i < len(points)-1; i++ {
		// Avoid a tiny last leg.
		if dists[i] >= next && total-dists[i] > spacing/2 {
			idxs = append(idxs, i)
			next = dists[i] + spacing
		}
	}
	idxs = append(idxs, len(points)-1)
	waypoints := make([]routes.LatLng, len(idxs))
	for i, idx := range idxs {
		waypoints[i] = routes.LatLng{Lat: points[idx].Lat, Lng: points[idx].Lng}
	}
	legs := make([]routes.Leg, len(idxs)-1)
	for i := range legs {
		legs[i] = routes.NewLeg(waypoints[i], waypoints[i+1], "", points[idxs[i]:idxs[i+1]+1], nil)
	}
	return waypoints, legs
}

// computeStats computes the statistics of a route's path.
func computeStats(route *routes.Route) {
	points, terrains := route.Path()
	route.Stats = geo.ComputeStats(points, terrains)
}
