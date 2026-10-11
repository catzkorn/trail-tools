package geo

import (
	"errors"
	"math"
	"strings"
)

// DecodePolyline decodes a Google encoded polyline with 5 digits of precision,
// as used by Strava. Elevations of the returned points are NaN.
func DecodePolyline(s string) ([]Point, error) {
	var points []Point
	var lat, lng int
	for i := 0; i < len(s); {
		var deltas [2]int
		for j := range deltas {
			var result, shift int
			for {
				if i >= len(s) {
					return nil, errors.New("truncated polyline")
				}
				b := int(s[i]) - 63
				i++
				if b < 0 {
					return nil, errors.New("invalid polyline character")
				}
				result |= (b & 0x1f) << shift
				shift += 5
				if b < 0x20 {
					break
				}
			}
			if result&1 != 0 {
				deltas[j] = ^(result >> 1)
			} else {
				deltas[j] = result >> 1
			}
		}
		lat += deltas[0]
		lng += deltas[1]
		points = append(points, Point{Lat: float64(lat) / 1e5, Lng: float64(lng) / 1e5, Ele: math.NaN()})
	}
	return points, nil
}

// EncodePolyline encodes points as a Google encoded polyline with 5 digits of
// precision.
func EncodePolyline(points []Point) string {
	var sb strings.Builder
	var prevLat, prevLng int
	for _, p := range points {
		lat := int(math.Round(p.Lat * 1e5))
		lng := int(math.Round(p.Lng * 1e5))
		encodeValue(&sb, lat-prevLat)
		encodeValue(&sb, lng-prevLng)
		prevLat, prevLng = lat, lng
	}
	return sb.String()
}

func encodeValue(sb *strings.Builder, v int) {
	v <<= 1
	if v < 0 {
		v = ^v
	}
	for v >= 0x20 {
		sb.WriteByte(byte((0x20 | (v & 0x1f)) + 63))
		v >>= 5
	}
	sb.WriteByte(byte(v + 63))
}
