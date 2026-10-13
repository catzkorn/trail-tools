package geo

import (
	"math"
)

const (
	// repeatSpacing is the distance between the samples of a track that are
	// checked for repeats.
	repeatSpacing = 10.0
	// repeatTolerance is how close a sample must be to an earlier part of the
	// track to repeat it, allowing for GPS noise and both sides of a trail.
	repeatTolerance = 15.0
	// repeatMinGap is how far along the track an earlier part must be, so
	// that the track doesn't repeat itself where it's simply continuing, or
	// on the legs of tight switchbacks.
	repeatMinGap = 300.0
	// repeatMinAlignment is the minimum |cos| of the angle between the
	// directions of the track and the earlier part it repeats: they run along
	// the same line, either way, rather than crossing it.
	repeatMinAlignment = 0.85
)

// RepeatedDistance returns the distance in meters of a track that retraces
// ground it covered earlier, in either direction. An out-and-back repeats
// about half its distance, a loop with a stem the stem's return, and a loop
// almost nothing. It's approximate: separate trails within about 15 m of each
// other count as the same.
func RepeatedDistance(points []Point) float64 {
	samples := resample(points, repeatSpacing)
	if len(samples) < 2 {
		return 0
	}
	// Index the samples by cell, with cells as wide as the tolerance, so
	// nearby samples are in the same or a neighboring cell.
	cells := map[[2]int][]int{}
	cellOf := func(s xy) [2]int {
		return [2]int{int(math.Floor(s.x / repeatTolerance)), int(math.Floor(s.y / repeatTolerance))}
	}
	direction := func(i int) (float64, float64) {
		a, b := samples[max(0, i-1)], samples[min(len(samples)-1, i+1)]
		dx, dy := b.x-a.x, b.y-a.y
		n := math.Hypot(dx, dy)
		if n == 0 {
			return 0, 0
		}
		return dx / n, dy / n
	}
	gap := int(repeatMinGap / repeatSpacing)
	repeated := 0
	for i, s := range samples {
		if i >= gap {
			// Earlier samples become candidates once they're far enough
			// behind.
			earlier := samples[i-gap]
			c := cellOf(earlier)
			cells[c] = append(cells[c], i-gap)
		}
		dx, dy := direction(i)
		c := cellOf(s)
	search:
		for x := c[0] - 1; x <= c[0]+1; x++ {
			for y := c[1] - 1; y <= c[1]+1; y++ {
				for _, j := range cells[[2]int{x, y}] {
					o := samples[j]
					if math.Hypot(s.x-o.x, s.y-o.y) > repeatTolerance {
						continue
					}
					ox, oy := direction(j)
					if math.Abs(dx*ox+dy*oy) >= repeatMinAlignment {
						repeated++
						break search
					}
				}
			}
		}
	}
	total := CumulativeDistances(points)[len(points)-1]
	return math.Min(total, float64(repeated)*repeatSpacing)
}

// xy is a position in meters on a local flat projection.
type xy struct{ x, y float64 }

// resample returns points spaced along a track, projected to meters around
// its first point.
func resample(points []Point, spacing float64) []xy {
	if len(points) == 0 {
		return nil
	}
	origin := points[0]
	cosLat := math.Cos(origin.Lat * math.Pi / 180)
	project := func(p Point) xy {
		return xy{
			x: (p.Lng - origin.Lng) * metersPerDegreeLat * cosLat,
			y: (p.Lat - origin.Lat) * metersPerDegreeLat,
		}
	}
	samples := []xy{project(points[0])}
	// The distance along the track to the next sample.
	next := spacing
	walked := 0.0
	for i := 1; i < len(points); i++ {
		d := Distance(points[i-1], points[i])
		for d > 0 && walked+d >= next {
			samples = append(samples, project(Interpolate(points[i-1], points[i], (next-walked)/d)))
			next += spacing
		}
		walked += d
	}
	return samples
}
