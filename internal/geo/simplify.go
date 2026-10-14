package geo

import "math"

// Clip returns the parts of a track inside bounds, each with at least two
// points. Lines crossing the edges of the bounds are cut at the edges, so
// parts of sparse tracks, such as simplified ones, aren't lost between points
// outside the bounds.
func Clip(points []Point, b Bounds) [][]Point {
	var parts [][]Point
	var part []Point
	for i := 1; i < len(points); i++ {
		from, to := points[i-1], points[i]
		t0, t1, ok := clipSegment(from, to, b)
		if !ok {
			continue
		}
		// A part ends where a line leaves the bounds, so a line continuing
		// one starts where the last one ended.
		if part == nil {
			part = []Point{Interpolate(from, to, t0)}
		}
		if t1 < 1 {
			parts = append(parts, append(part, Interpolate(from, to, t1)))
			part = nil
			continue
		}
		part = append(part, to)
	}
	if part != nil {
		parts = append(parts, part)
	}
	return parts
}

// clipSegment returns the fractions of the way from a to b where the line
// between them enters and leaves bounds, and whether it's inside the bounds
// at all (the Liang-Barsky algorithm).
func clipSegment(a, b Point, bounds Bounds) (t0, t1 float64, ok bool) {
	t0, t1 = 0, 1
	dLng, dLat := b.Lng-a.Lng, b.Lat-a.Lat
	// For each edge, the change towards it and the distance to it.
	for _, edge := range [4][2]float64{
		{-dLng, a.Lng - bounds.MinLng},
		{dLng, bounds.MaxLng - a.Lng},
		{-dLat, a.Lat - bounds.MinLat},
		{dLat, bounds.MaxLat - a.Lat},
	} {
		change, distance := edge[0], edge[1]
		switch {
		case change == 0:
			// Parallel to the edge, and outside it.
			if distance < 0 {
				return 0, 0, false
			}
		case change < 0:
			// Entering the bounds across the edge.
			t0 = max(t0, distance/change)
		default:
			// Leaving the bounds across the edge.
			t1 = min(t1, distance/change)
		}
		if t0 > t1 {
			return 0, 0, false
		}
	}
	return t0, t1, true
}

// Simplify removes points of a track that are within tolerance meters of the
// line through their neighbors (the Douglas-Peucker algorithm).
func Simplify(points []Point, tolerance float64) []Point {
	if len(points) < 3 {
		return points
	}
	worst, worstIndex := 0.0, 0
	last := len(points) - 1
	for i := 1; i < last; i++ {
		if d := SegmentDistance(points[i], points[0], points[last]); d > worst {
			worst, worstIndex = d, i
		}
	}
	if worst <= tolerance {
		return []Point{points[0], points[last]}
	}
	first := Simplify(points[:worstIndex+1], tolerance)
	// Capped, so appending to it doesn't overwrite the points.
	first = first[: len(first)-1 : len(first)-1]
	return append(first, Simplify(points[worstIndex:], tolerance)...)
}

// SegmentDistance returns the distance in meters from p to the segment from a
// to b, in a flat projection around p, which is accurate over the short
// distances it is used for.
func SegmentDistance(p, a, b Point) float64 {
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
