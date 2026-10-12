package trip

import (
	"errors"
	"math"
	"math/rand/v2"
	"strings"
	"testing"
)

// syntheticProfile samples an elevation function of the distance in
// kilometers every 10 meters up to km kilometers, with up to noise meters
// of random noise.
func syntheticProfile(km, noise float64, ele func(km float64) float64) (dists, eles []float64) {
	r := rand.New(rand.NewPCG(1, 2))
	for d := 0.0; d <= km*1000; d += 10 {
		dists = append(dists, d)
		eles = append(eles, ele(d/1000)+noise*(2*r.Float64()-1))
	}
	return dists, eles
}

// linear interpolates between the elevations at kilometers, given as pairs
// of kilometer and elevation.
func linear(points ...float64) func(km float64) float64 {
	return func(km float64) float64 {
		for i := 2; i < len(points); i += 2 {
			if km <= points[i] {
				f := (km - points[i-2]) / (points[i] - points[i-2])
				return points[i-1] + f*(points[i+1]-points[i-1])
			}
		}
		return points[len(points)-1]
	}
}

type wantSegment struct {
	kind segmentKind
	// The name, or its start if it ends with "…".
	name string
	// The kilometer where the segment ends.
	endKm float64
}

func TestSuggestSegments(t *testing.T) {
	tests := []struct {
		name  string
		km    float64
		noise float64
		ele   func(km float64) float64
		want  []wantSegment
	}{
		{
			name:  "up and down",
			km:    20,
			noise: 3,
			ele:   linear(0, 500, 10, 1500, 20, 500),
			want: []wantSegment{
				{segmentClimb, "Climb to 1,500 m", 10},
				{segmentDescent, "Final descent to 500 m", 20},
			},
		},
		{
			name:  "climb, ridge, dip, summit and descent",
			km:    25,
			noise: 4,
			ele: func(km float64) float64 {
				if km > 6 && km < 12 {
					// Bumps of 80 m along the ridge.
					return 1400 + 40*math.Sin(2*math.Pi*(km-6)/1.5)
				}
				return linear(0, 600, 6, 1400, 12, 1400, 14, 1200, 17, 1900, 25, 600)(km)
			},
			want: []wantSegment{
				{segmentClimb, "First climb to 1,4…", 6},
				{segmentRolling, "Rolling ridge, 6 km", 12},
				{segmentDescent, "Descent to 1,2…", 14},
				{segmentClimb, "Climb to the high point, 1,900 m", 17},
				{segmentDescent, "Final descent to 600 m", 25},
			},
		},
		{
			name:  "valley start",
			km:    18,
			noise: 2,
			ele:   linear(0, 400, 4, 410, 9, 1200, 18, 400),
			want: []wantSegment{
				{segmentFlat, "Flat stretch, 4 km", 4},
				{segmentClimb, "Climb to 1,200 m", 9},
				{segmentDescent, "Final descent to 400 m", 18},
			},
		},
		{
			name:  "noisy flat",
			km:    15,
			noise: 3,
			ele:   linear(0, 100, 15, 105),
			want: []wantSegment{
				{segmentFlat, "Flat stretch, 15 km", 15},
			},
		},
		{
			name:  "short bumps are merged",
			km:    16,
			noise: 2,
			// A 25 m dip in the climb, and a 20 m bump in the descent.
			ele: linear(0, 500, 4, 900, 4.3, 875, 8, 1300, 11, 1000, 11.3, 1020, 16, 500),
			want: []wantSegment{
				{segmentClimb, "Climb to 1,300 m", 8},
				{segmentDescent, "Final descent to 500 m", 16},
			},
		},
		{
			name:  "rolling terrain low down",
			km:    20,
			noise: 2,
			ele: func(km float64) float64 {
				if km < 12 {
					return 300 + 35*math.Sin(2*math.Pi*km/2)
				}
				return linear(12, 300, 16, 1100, 20, 1000)(km)
			},
			want: []wantSegment{
				// The last trough is at 11.5 km.
				{segmentRolling, "Rolling terrain, …", 12},
				{segmentClimb, "Climb to 1,100 m", 16},
				{segmentDescent, "Final descent to 1,000 m", 20},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dists, eles := syntheticProfile(tt.km, tt.noise, tt.ele)
			segs, err := suggestSegments(dists, eles)
			if err != nil {
				t.Fatal(err)
			}
			if len(segs) != len(tt.want) {
				t.Fatalf("got %d segments, want %d:\n%s", len(segs), len(tt.want), describeSegments(segs))
			}
			for i, s := range segs {
				w := tt.want[i]
				if s.kind != w.kind {
					t.Errorf("segment %d is a %v, want a %v:\n%s", i, s.kind, w.kind, describeSegments(segs))
				}
				if prefix, ok := strings.CutSuffix(w.name, "…"); ok && !strings.HasPrefix(s.name, prefix) || !ok && s.name != w.name {
					t.Errorf("segment %d is named %q, want %q", i, s.name, w.name)
				}
				if math.Abs(s.endMeters/1000-w.endKm) > 0.6 {
					t.Errorf("segment %d ends at %.1f km, want %.1f km", i, s.endMeters/1000, w.endKm)
				}
			}
			if segs[0].startMeters != 0 || segs[len(segs)-1].endMeters != tt.km*1000 {
				t.Errorf("segments cover %.0f to %.0f m, want the whole track", segs[0].startMeters, segs[len(segs)-1].endMeters)
			}
			for i := 1; i < len(segs); i++ {
				if segs[i].startMeters != segs[i-1].endMeters {
					t.Errorf("segment %d starts at %.0f m, but the one before ends at %.0f m", i, segs[i].startMeters, segs[i-1].endMeters)
				}
			}
		})
	}
}

func TestSuggestSegmentsLimitsSections(t *testing.T) {
	// Twelve climbs and descents of 400 m.
	dists, eles := syntheticProfile(48, 2, func(km float64) float64 {
		return 1000 + 200*math.Sin(2*math.Pi*km/4)
	})
	segs, err := suggestSegments(dists, eles)
	if err != nil {
		t.Fatal(err)
	}
	if len(segs) > maxSuggestedSections {
		t.Errorf("got %d segments, want at most %d:\n%s", len(segs), maxSuggestedSections, describeSegments(segs))
	}
}

func TestSuggestSegmentsWithoutProfile(t *testing.T) {
	dists := []float64{0, 1000, 2000}
	for _, eles := range [][]float64{
		{math.NaN(), math.NaN(), math.NaN()},
		{100, math.NaN(), math.NaN()},
	} {
		if _, err := suggestSegments(dists, eles); !errors.Is(err, errNoProfile) {
			t.Errorf("suggestSegments(%v) = %v, want errNoProfile", eles, err)
		}
	}
	if _, err := suggestSegments([]float64{0, 500}, []float64{100, 120}); !errors.Is(err, errNoProfile) {
		t.Errorf("suggestSegments on 500 m = %v, want errNoProfile", err)
	}
}

func TestFormatElevation(t *testing.T) {
	tests := map[float64]string{
		0:       "0 m",
		94:      "90 m",
		1447:    "1,450 m",
		12345:   "12,350 m",
		-23:     "-20 m",
		1234567: "1,234,570 m",
	}
	for meters, want := range tests {
		if got := formatElevation(meters); got != want {
			t.Errorf("formatElevation(%v) = %q, want %q", meters, got, want)
		}
	}
}

func TestFormatKilometers(t *testing.T) {
	tests := map[float64]string{
		20:    "0.1 km",
		1000:  "1 km",
		2460:  "2.5 km",
		5900:  "6 km",
		12345: "12 km",
	}
	for meters, want := range tests {
		if got := formatKilometers(meters); got != want {
			t.Errorf("formatKilometers(%v) = %q, want %q", meters, got, want)
		}
	}
}

func describeSegments(segs []*segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteString("  ")
		b.WriteString(s.kind.String())
		b.WriteString(" ")
		b.WriteString(formatKilometers(s.startMeters))
		b.WriteString(" to ")
		b.WriteString(formatKilometers(s.endMeters))
		b.WriteString(": ")
		b.WriteString(s.name)
		b.WriteString("\n")
	}
	return b.String()
}
