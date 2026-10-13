package geo

import (
	"math"
	"strings"
)

// Terrain is a coarse classification of the surface underfoot.
type Terrain string

// Terrain types. The zero value means unknown.
const (
	TerrainUnknown   Terrain = ""
	TerrainPaved     Terrain = "paved"
	TerrainGravel    Terrain = "gravel"
	TerrainTrail     Terrain = "trail"
	TerrainTechnical Terrain = "technical"
)

// ClassifyTerrain classifies an OpenStreetMap way by its tags.
func ClassifyTerrain(tags map[string]string) Terrain {
	switch tags["sac_scale"] {
	case "mountain_hiking", "demanding_mountain_hiking", "alpine_hiking",
		"demanding_alpine_hiking", "difficult_alpine_hiking":
		return TerrainTechnical
	}
	switch tags["surface"] {
	case "rock", "bare_rock", "scree", "stone", "boulders":
		return TerrainTechnical
	case "asphalt", "concrete", "concrete:plates", "concrete:lanes", "paved",
		"paving_stones", "sett", "cobblestone", "unhewn_cobblestone", "metal", "wood",
		"chipseal", "brick", "bricks":
		return TerrainPaved
	case "gravel", "fine_gravel", "compacted", "pebblestone", "shells":
		return TerrainGravel
	case "dirt", "ground", "earth", "grass", "mud", "sand", "woodchips", "unpaved",
		"grass_paver", "forest_floor", "soil", "clay", "snow", "ice":
		return TerrainTrail
	}
	switch tags["highway"] {
	case "path", "bridleway":
		return TerrainTrail
	case "track":
		switch tags["tracktype"] {
		case "grade1":
			return TerrainPaved
		case "grade3", "grade4", "grade5":
			return TerrainTrail
		}
		return TerrainGravel
	case "":
		return TerrainUnknown
	}
	// Roads, footways, cycleways, steps and pedestrian areas without a surface
	// tag are almost always paved.
	return TerrainPaved
}

// ParseTags parses space separated key=value tags.
func ParseTags(s string) map[string]string {
	tags := map[string]string{}
	for _, field := range strings.Fields(s) {
		k, v, ok := strings.Cut(field, "=")
		if ok {
			tags[k] = v
		}
	}
	return tags
}

// elevationThreshold is the hysteresis in meters applied when accumulating
// elevation gain and loss, to filter out noise.
const elevationThreshold = 4.0

// ElevationGainLoss computes the total elevation gain and loss of a track in
// meters, ignoring changes smaller than a small threshold to filter noise.
// Points with unknown elevation are skipped.
func ElevationGainLoss(elevations []float64) (gain, loss float64) {
	ref := math.NaN()
	for _, ele := range elevations {
		if math.IsNaN(ele) {
			continue
		}
		if math.IsNaN(ref) {
			ref = ele
			continue
		}
		switch diff := ele - ref; {
		case diff >= elevationThreshold:
			gain += diff
			ref = ele
		case diff <= -elevationThreshold:
			loss -= diff
			ref = ele
		}
	}
	// Count the change since the last reference point, so the totals add up
	// to the difference between the start and end.
	for i := len(elevations) - 1; i >= 0; i-- {
		if math.IsNaN(elevations[i]) {
			continue
		}
		if diff := elevations[i] - ref; diff > 0 {
			gain += diff
		} else {
			loss -= diff
		}
		break
	}
	return gain, loss
}

// gradeWindowMeters is the distance over which maximum grades are measured.
const gradeWindowMeters = 200.0

// MaxGrades returns the steepest uphill and downhill grade in percent,
// measured over a sliding window of 200 meters. The downhill grade is returned
// as a positive number.
func MaxGrades(dists, elevations []float64) (climb, descent float64) {
	j := 0
	for i := range dists {
		// Advance j to the last point that is at least a window behind i.
		for j+1 < i && dists[i]-dists[j+1] >= gradeWindowMeters {
			j++
		}
		run := dists[i] - dists[j]
		if run < gradeWindowMeters || math.IsNaN(elevations[i]) || math.IsNaN(elevations[j]) {
			continue
		}
		grade := (elevations[i] - elevations[j]) / run * 100
		climb = math.Max(climb, grade)
		descent = math.Max(descent, -grade)
	}
	return climb, descent
}

// TrackStats are statistics computed from a track.
type TrackStats struct {
	DistanceMeters      float64
	ElevationGainMeters float64
	ElevationLossMeters float64
	MinElevationMeters  float64
	MaxElevationMeters  float64
	MaxClimbGrade       float64
	MaxDescentGrade     float64
	// Distance in meters by terrain type.
	TerrainMeters map[Terrain]float64
	// Distance in meters that retraces ground covered earlier, see
	// RepeatedDistance. Nil for stats stored before it was computed.
	RepeatedMeters *float64 `json:",omitempty"`
}

// ComputeStats computes statistics for a track. terrains may be nil, or have
// one entry per point, where terrains[i] is the terrain of the segment from
// point i to point i+1.
func ComputeStats(points []Point, terrains []Terrain) TrackStats {
	dists := CumulativeDistances(points)
	elevations := make([]float64, len(points))
	stats := TrackStats{
		TerrainMeters:      map[Terrain]float64{},
		MinElevationMeters: math.Inf(1),
		MaxElevationMeters: math.Inf(-1),
	}
	for i, p := range points {
		elevations[i] = p.Ele
		if !math.IsNaN(p.Ele) {
			stats.MinElevationMeters = math.Min(stats.MinElevationMeters, p.Ele)
			stats.MaxElevationMeters = math.Max(stats.MaxElevationMeters, p.Ele)
		}
		if i > 0 {
			terrain := TerrainUnknown
			if terrains != nil {
				terrain = terrains[i-1]
			}
			stats.TerrainMeters[terrain] += dists[i] - dists[i-1]
		}
	}
	if math.IsInf(stats.MinElevationMeters, 0) {
		stats.MinElevationMeters, stats.MaxElevationMeters = 0, 0
	}
	if len(dists) > 0 {
		stats.DistanceMeters = dists[len(dists)-1]
	}
	stats.ElevationGainMeters, stats.ElevationLossMeters = ElevationGainLoss(elevations)
	stats.MaxClimbGrade, stats.MaxDescentGrade = MaxGrades(dists, elevations)
	repeated := RepeatedDistance(points)
	stats.RepeatedMeters = &repeated
	return stats
}
