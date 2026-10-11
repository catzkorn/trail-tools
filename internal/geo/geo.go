// Package geo contains geometry helpers for working with GPS tracks: distances,
// encoded polylines, GPX files, elevation statistics and terrain.
package geo

import (
	"math"
)

const earthRadiusMeters = 6371008.8

// Point is a WGS84 coordinate with an elevation.
type Point struct {
	Lat float64
	Lng float64
	// Elevation in meters. NaN if unknown.
	Ele float64
}

// Distance returns the great circle distance between two points in meters.
func Distance(a, b Point) float64 {
	lat1 := a.Lat * math.Pi / 180
	lat2 := b.Lat * math.Pi / 180
	dLat := lat2 - lat1
	dLng := (b.Lng - a.Lng) * math.Pi / 180
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusMeters * math.Asin(math.Min(1, math.Sqrt(h)))
}

// CumulativeDistances returns the distance from the first point to each point
// along the track, in meters.
func CumulativeDistances(points []Point) []float64 {
	dists := make([]float64, len(points))
	for i := 1; i < len(points); i++ {
		dists[i] = dists[i-1] + Distance(points[i-1], points[i])
	}
	return dists
}

// Interpolate returns the point a fraction f of the way from a to b.
func Interpolate(a, b Point, f float64) Point {
	return Point{
		Lat: a.Lat + (b.Lat-a.Lat)*f,
		Lng: a.Lng + (b.Lng-a.Lng)*f,
		Ele: a.Ele + (b.Ele-a.Ele)*f,
	}
}

// Densify inserts points between a and b so that no two consecutive points are
// further apart than spacing meters. The result includes both a and b.
func Densify(a, b Point, spacing float64) []Point {
	n := max(int(math.Ceil(Distance(a, b)/spacing)), 1)
	points := make([]Point, 0, n+1)
	for i := 0; i <= n; i++ {
		points = append(points, Interpolate(a, b, float64(i)/float64(n)))
	}
	return points
}

// Bounds is a bounding box.
type Bounds struct {
	MinLat, MinLng, MaxLat, MaxLng float64
}

// BoundsOf returns the bounding box of the points. ok is false if there are no
// points.
func BoundsOf(points []Point) (b Bounds, ok bool) {
	if len(points) == 0 {
		return Bounds{}, false
	}
	b = Bounds{MinLat: 90, MinLng: 180, MaxLat: -90, MaxLng: -180}
	for _, p := range points {
		b.MinLat = math.Min(b.MinLat, p.Lat)
		b.MinLng = math.Min(b.MinLng, p.Lng)
		b.MaxLat = math.Max(b.MaxLat, p.Lat)
		b.MaxLng = math.Max(b.MaxLng, p.Lng)
	}
	return b, true
}

// Downsample returns at most limit indexes into a slice of length n, evenly
// spaced and always including the first and last index.
func Downsample(n, limit int) []int {
	if n <= 0 {
		return nil
	}
	limit = max(limit, 2)
	if n <= limit {
		idxs := make([]int, n)
		for i := range idxs {
			idxs[i] = i
		}
		return idxs
	}
	idxs := make([]int, 0, limit)
	step := float64(n-1) / float64(limit-1)
	last := -1
	for i := range limit {
		idx := int(math.Round(float64(i) * step))
		if idx != last {
			idxs = append(idxs, idx)
			last = idx
		}
	}
	return idxs
}

// NearestIndex returns the index of the point closest to p.
func NearestIndex(points []Point, p Point) (int, float64) {
	best, bestDist := -1, math.Inf(1)
	for i, q := range points {
		if d := Distance(p, q); d < bestDist {
			best, bestDist = i, d
		}
	}
	return best, bestDist
}
