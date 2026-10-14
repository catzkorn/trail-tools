package geo_test

import (
	"math"
	"slices"
	"testing"

	"github.com/catzkorn/trail-tools/internal/geo"
)

func TestClip(t *testing.T) {
	t.Parallel()
	bounds := geo.Bounds{MinLat: 0, MinLng: 0, MaxLat: 1, MaxLng: 1}
	for _, tt := range []struct {
		name   string
		points []geo.Point
		want   [][]geo.Point
	}{
		{
			name:   "inside",
			points: []geo.Point{{Lat: 0.2, Lng: 0.2}, {Lat: 0.5, Lng: 0.5}, {Lat: 0.8, Lng: 0.2}},
			want:   [][]geo.Point{{{Lat: 0.2, Lng: 0.2}, {Lat: 0.5, Lng: 0.5}, {Lat: 0.8, Lng: 0.2}}},
		},
		{
			name:   "crossing without points inside",
			points: []geo.Point{{Lat: 0.5, Lng: -1}, {Lat: 0.5, Lng: 2}},
			want:   [][]geo.Point{{{Lat: 0.5, Lng: 0}, {Lat: 0.5, Lng: 1}}},
		},
		{
			name:   "leaving and coming back",
			points: []geo.Point{{Lat: 0.5, Lng: 0.5}, {Lat: 1.5, Lng: 0.5}, {Lat: 0.5, Lng: 0.75}, {Lat: 0.25, Lng: 0.75}},
			want: [][]geo.Point{
				{{Lat: 0.5, Lng: 0.5}, {Lat: 1, Lng: 0.5}},
				{{Lat: 1, Lng: 0.625}, {Lat: 0.5, Lng: 0.75}, {Lat: 0.25, Lng: 0.75}},
			},
		},
		{
			name:   "passing by",
			points: []geo.Point{{Lat: -1, Lng: 0.5}, {Lat: 0.5, Lng: -1}, {Lat: 2, Lng: -1}},
		},
		{
			name:   "one point",
			points: []geo.Point{{Lat: 0.5, Lng: 0.5}},
		},
	} {
		got := geo.Clip(tt.points, bounds)
		if !slices.EqualFunc(got, tt.want, func(a, b []geo.Point) bool {
			return slices.EqualFunc(a, b, func(p, q geo.Point) bool {
				return math.Abs(p.Lat-q.Lat) < 1e-12 && math.Abs(p.Lng-q.Lng) < 1e-12
			})
		}) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSimplify(t *testing.T) {
	t.Parallel()
	// Two straight lines to a point 30 meters east of the ends, with less
	// than a meter of noise.
	var points []geo.Point
	for i := range 101 {
		east := 30*(1-math.Abs(float64(i-50))/50) + 0.3*float64(i%2)
		points = append(points, geo.Point{Lat: float64(i) * 10 / 111320, Lng: east / 111320})
	}
	original := slices.Clone(points)
	got := geo.Simplify(points, 2)
	if len(got) != 3 || got[0] != points[0] || got[1] != points[50] || got[2] != points[100] {
		t.Errorf("got %v, want the ends and the bump", got)
	}
	if !slices.Equal(points, original) {
		t.Error("the points were changed")
	}
}
