package track

import (
	routesv1 "github.com/johanbrandhorst/trail-tools/internal/gen/routes/v1"
	"github.com/johanbrandhorst/trail-tools/internal/geo"
)

// Zone is a circle, like one around the user's home, within which tracks
// and photos are hidden on public pages.
type Zone struct {
	Center       geo.Point
	RadiusMeters float64
}

// InZones reports whether a point is within any of the zones.
func InZones(p geo.Point, zones []Zone) bool {
	for _, z := range zones {
		if geo.Distance(p, z.Center) < z.RadiusMeters {
			return true
		}
	}
	return false
}

// PublicPoints returns the track as route points without those within the
// zones, and the indexes of the returned points that start a new piece of
// the track after a hidden stretch, which mustn't be joined to the point
// before. Distances stay along the whole track.
func (t *Track) PublicPoints(zones []Zone) (points []*routesv1.RoutePoint, breaks []int32) {
	all := t.RoutePoints()
	if len(zones) == 0 {
		return all, nil
	}
	hidden := false
	for i, p := range t.Points {
		if InZones(p, zones) {
			hidden = true
			continue
		}
		if hidden && len(points) > 0 {
			breaks = append(breaks, int32(len(points)))
		}
		hidden = false
		points = append(points, all[i])
	}
	return points, breaks
}
