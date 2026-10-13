package geo

import (
	"math"
	"strconv"
	"strings"
)

// FeatureKind is the kind of a point of interest for trail running.
type FeatureKind string

// Feature kinds. The zero value is not a point of interest.
const (
	FeatureNone      FeatureKind = ""
	FeaturePeak      FeatureKind = "peak"
	FeatureWater     FeatureKind = "water"
	FeatureParking   FeatureKind = "parking"
	FeatureViewpoint FeatureKind = "viewpoint"
	FeatureShelter   FeatureKind = "shelter"
	FeatureToilets   FeatureKind = "toilets"
	FeatureSaddle    FeatureKind = "saddle"
	FeatureTrailhead FeatureKind = "trailhead"
)

// ClassifyFeature returns the kind of point of interest an OpenStreetMap
// node or way is, from its tags.
func ClassifyFeature(tags map[string]string) FeatureKind {
	switch {
	case tags["natural"] == "peak" || tags["natural"] == "volcano":
		return FeaturePeak
	case tags["amenity"] == "drinking_water" || tags["natural"] == "spring" || tags["man_made"] == "water_tap":
		return FeatureWater
	case tags["amenity"] == "parking":
		return FeatureParking
	case tags["tourism"] == "viewpoint":
		return FeatureViewpoint
	case tags["amenity"] == "shelter" || tags["tourism"] == "alpine_hut" || tags["tourism"] == "wilderness_hut":
		return FeatureShelter
	case tags["amenity"] == "toilets":
		return FeatureToilets
	case tags["natural"] == "saddle" || tags["mountain_pass"] == "yes":
		return FeatureSaddle
	case tags["highway"] == "trailhead":
		return FeatureTrailhead
	}
	return FeatureNone
}

// ParseElevation parses an OpenStreetMap ele tag, in meters, such as "1234"
// or "1234 m".
func ParseElevation(s string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(s), " m"), 64)
	return v, err == nil && !math.IsNaN(v) && !math.IsInf(v, 0)
}
