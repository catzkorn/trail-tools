package track

import (
	"slices"
	"testing"

	"github.com/johanbrandhorst/trail-tools/internal/geo"
)

func TestPublicPoints(t *testing.T) {
	t.Parallel()
	// A track heading north, a point about every 111 meters.
	var points []geo.Point
	for i := range 11 {
		points = append(points, geo.Point{Lat: float64(i) * 0.001})
	}
	tr := &Track{Points: points, Dists: geo.CumulativeDistances(points)}
	for name, tt := range map[string]struct {
		zones      []Zone
		wantLats   []float64
		wantBreaks []int32
	}{
		"no zones": {nil, []float64{0, 0.001, 0.002, 0.003, 0.004, 0.005, 0.006, 0.007, 0.008, 0.009, 0.01}, nil},
		// Hiding the start, around the first two points.
		"at the start": {[]Zone{{Center: points[0], RadiusMeters: 150}}, []float64{0.002, 0.003, 0.004, 0.005, 0.006, 0.007, 0.008, 0.009, 0.01}, nil},
		// Hiding the middle splits the track in two.
		"in the middle": {[]Zone{{Center: points[5], RadiusMeters: 150}}, []float64{0, 0.001, 0.002, 0.003, 0.007, 0.008, 0.009, 0.01}, []int32{4}},
		"everywhere":    {[]Zone{{Center: points[5], RadiusMeters: 5000}}, nil, nil},
	} {
		got, breaks := tr.PublicPoints(tt.zones)
		var lats []float64
		for _, p := range got {
			lats = append(lats, float64(int(p.GetLat()*1000+0.5))/1000)
		}
		if !slices.Equal(lats, tt.wantLats) || !slices.Equal(breaks, tt.wantBreaks) {
			t.Errorf("%s: got %v with breaks %v, want %v with breaks %v", name, lats, breaks, tt.wantLats, tt.wantBreaks)
		}
		for i, p := range got {
			if i > 0 && p.GetDistanceMeters() <= got[i-1].GetDistanceMeters() {
				t.Errorf("%s: distances aren't along the whole track: %v", name, got)
			}
		}
	}
}
