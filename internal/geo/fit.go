package geo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"time"
	"unicode/utf8"
)

// CoursePointType is the kind of a course point, as numbered in the FIT
// profile. Only the kinds all devices know are used.
type CoursePointType uint8

// Course point types.
const (
	CoursePointGeneric CoursePointType = 0
	CoursePointSummit  CoursePointType = 1
	CoursePointWater   CoursePointType = 3
	CoursePointFood    CoursePointType = 4
)

// CoursePoint is a named point along a course, like an aid station, which
// devices list and announce on the way.
type CoursePoint struct {
	Name string
	// The distance along the course in meters.
	DistanceMeters float64
	// The planned elapsed seconds when the point is reached.
	ElapsedSeconds float64
	Type           CoursePointType
}

// FITCourse is a course to write as a FIT file.
type FITCourse struct {
	Name   string
	Points []Point
	// The planned elapsed seconds at each point, which devices race as a
	// virtual partner. They must not decrease.
	ElapsedSeconds []float64
	CoursePoints   []CoursePoint
	// When the course starts. Devices only use the times relative to it.
	Start time.Time
}

// fitEpoch is the start of FIT timestamps.
var fitEpoch = time.Date(1989, 12, 31, 0, 0, 0, 0, time.UTC)

// FIT base types.
const (
	fitEnum   = 0x00
	fitUint8  = 0x02
	fitString = 0x07
	fitUint16 = 0x84
	fitSint32 = 0x85
	fitUint32 = 0x86
)

// FIT global message numbers.
const (
	fitFileID      = 0
	fitLap         = 19
	fitRecord      = 20
	fitEvent       = 21
	fitCourse      = 31
	fitCoursePoint = 32
)

const (
	// fitNameSize is the size of course names, including the terminating
	// zero.
	fitNameSize = 32
	// fitPointNameSize is the size of course point names, including the
	// terminating zero. Devices show about this many characters.
	fitPointNameSize = 16
	// fitInvalidUint16 marks an unknown uint16 value, like an altitude.
	fitInvalidUint16 = 0xFFFF
)

type fitField struct {
	num, size, baseType uint8
}

// fitWriter writes FIT messages to a buffer.
type fitWriter struct {
	buf bytes.Buffer
}

// define writes a definition message for a local message type.
func (f *fitWriter) define(local uint8, global uint16, fields ...fitField) {
	f.buf.WriteByte(0x40 | local)
	// Reserved, then little endian.
	f.buf.Write([]byte{0, 0})
	_ = binary.Write(&f.buf, binary.LittleEndian, global)
	f.buf.WriteByte(uint8(len(fields)))
	for _, field := range fields {
		f.buf.Write([]byte{field.num, field.size, field.baseType})
	}
}

// data writes a data message for a local message type. The values must
// match its definition: fixed size integers, or strings padded to their
// field size with fitString.
func (f *fitWriter) data(local uint8, values ...any) {
	f.buf.WriteByte(local)
	for _, v := range values {
		_ = binary.Write(&f.buf, binary.LittleEndian, v)
	}
}

// fitStringValue returns s as a zero terminated string of size bytes,
// truncated at a character boundary if it is too long.
func fitStringValue(s string, size int) []byte {
	b := make([]byte, size)
	for len(s) > size-1 {
		_, n := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-n]
	}
	copy(b, s)
	return b
}

func fitTime(t time.Time) uint32 {
	return uint32(t.Sub(fitEpoch) / time.Second)
}

func semicircles(degrees float64) int32 {
	return int32(math.Round(degrees * (1 << 31) / 180))
}

func fitAltitude(ele float64) uint16 {
	if math.IsNaN(ele) {
		return fitInvalidUint16
	}
	return uint16(math.Round((min(max(ele, -500), 12000) + 500) * 5))
}

// WriteFITCourse encodes a course as a FIT course file, which Garmin and
// other devices follow with turn by turn guidance, a virtual partner and
// alerts at its course points.
func WriteFITCourse(w io.Writer, c *FITCourse) error {
	if len(c.Points) < 2 {
		return errors.New("a course needs at least two points")
	}
	if len(c.ElapsedSeconds) != len(c.Points) {
		return errors.New("a course needs an elapsed time for each point")
	}
	dists := CumulativeDistances(c.Points)
	timeAt := func(i int) uint32 {
		return fitTime(c.Start.Add(time.Duration(c.ElapsedSeconds[i] * float64(time.Second))))
	}
	first, last := c.Points[0], c.Points[len(c.Points)-1]
	start, end := timeAt(0), timeAt(len(c.Points)-1)
	elevations := make([]float64, len(c.Points))
	for i, p := range c.Points {
		elevations[i] = p.Ele
	}
	gain, loss := ElevationGainLoss(elevations)
	elapsedMillis := uint32(math.Round((c.ElapsedSeconds[len(c.Points)-1] - c.ElapsedSeconds[0]) * 1000))

	var f fitWriter
	f.define(0, fitFileID,
		fitField{0, 1, fitEnum},   // Type.
		fitField{1, 2, fitUint16}, // Manufacturer.
		fitField{2, 2, fitUint16}, // Product.
		fitField{4, 4, fitUint32}, // Time created.
	)
	// A course file from a development manufacturer.
	f.data(0, uint8(6), uint16(255), uint16(0), start)
	f.define(1, fitCourse,
		fitField{4, 1, fitEnum},             // Sport.
		fitField{5, fitNameSize, fitString}, // Name.
	)
	f.data(1, uint8(1), fitStringValue(c.Name, fitNameSize))
	f.define(2, fitLap,
		fitField{253, 4, fitUint32}, // Timestamp.
		fitField{2, 4, fitUint32},   // Start time.
		fitField{3, 4, fitSint32},   // Start latitude.
		fitField{4, 4, fitSint32},   // Start longitude.
		fitField{5, 4, fitSint32},   // End latitude.
		fitField{6, 4, fitSint32},   // End longitude.
		fitField{7, 4, fitUint32},   // Total elapsed time in milliseconds.
		fitField{8, 4, fitUint32},   // Total timer time in milliseconds.
		fitField{9, 4, fitUint32},   // Total distance in centimeters.
		fitField{21, 2, fitUint16},  // Total ascent in meters.
		fitField{22, 2, fitUint16},  // Total descent in meters.
	)
	f.data(2, end, start,
		semicircles(first.Lat), semicircles(first.Lng), semicircles(last.Lat), semicircles(last.Lng),
		elapsedMillis, elapsedMillis, uint32(math.Round(dists[len(dists)-1]*100)),
		uint16(math.Round(gain)), uint16(math.Round(loss)))
	f.define(3, fitEvent,
		fitField{253, 4, fitUint32}, // Timestamp.
		fitField{0, 1, fitEnum},     // Event.
		fitField{1, 1, fitEnum},     // Event type.
		fitField{4, 1, fitUint8},    // Event group.
	)
	// The timer starts at the start, and stops at the finish.
	f.data(3, start, uint8(0), uint8(0), uint8(0))
	f.define(4, fitRecord,
		fitField{253, 4, fitUint32}, // Timestamp.
		fitField{0, 4, fitSint32},   // Latitude.
		fitField{1, 4, fitSint32},   // Longitude.
		fitField{2, 2, fitUint16},   // Altitude.
		fitField{5, 4, fitUint32},   // Distance in centimeters.
	)
	for i, p := range c.Points {
		f.data(4, timeAt(i), semicircles(p.Lat), semicircles(p.Lng), fitAltitude(p.Ele), uint32(math.Round(dists[i]*100)))
	}
	f.data(3, end, uint8(0), uint8(9), uint8(0))
	if len(c.CoursePoints) > 0 {
		f.define(5, fitCoursePoint,
			fitField{1, 4, fitUint32},                // Timestamp.
			fitField{2, 4, fitSint32},                // Latitude.
			fitField{3, 4, fitSint32},                // Longitude.
			fitField{4, 4, fitUint32},                // Distance in centimeters.
			fitField{5, 1, fitEnum},                  // Type.
			fitField{6, fitPointNameSize, fitString}, // Name.
		)
	}
	for _, cp := range c.CoursePoints {
		d := min(max(cp.DistanceMeters, 0), dists[len(dists)-1])
		p := pointAtDistance(c.Points, dists, d)
		at := fitTime(c.Start.Add(time.Duration(cp.ElapsedSeconds * float64(time.Second))))
		f.data(5, at, semicircles(p.Lat), semicircles(p.Lng), uint32(math.Round(d*100)), uint8(cp.Type), fitStringValue(cp.Name, fitPointNameSize))
	}

	header := make([]byte, 14)
	header[0] = 14
	// Protocol version 2.0, profile version 21.00.
	header[1] = 0x20
	binary.LittleEndian.PutUint16(header[2:], 2100)
	binary.LittleEndian.PutUint32(header[4:], uint32(f.buf.Len()))
	copy(header[8:], ".FIT")
	binary.LittleEndian.PutUint16(header[12:], fitCRC(0, header[:12]))
	crc := fitCRC(fitCRC(0, header), f.buf.Bytes())
	file := append(header, f.buf.Bytes()...)
	file = binary.LittleEndian.AppendUint16(file, crc)
	_, err := w.Write(file)
	return err
}

// pointAtDistance returns the point at a distance along the points.
func pointAtDistance(points []Point, dists []float64, d float64) Point {
	for i := 1; i < len(points); i++ {
		if dists[i] < d {
			continue
		}
		segment := dists[i] - dists[i-1]
		if segment <= 0 {
			return points[i]
		}
		return Interpolate(points[i-1], points[i], (d-dists[i-1])/segment)
	}
	return points[len(points)-1]
}

var fitCRCTable = [16]uint16{
	0x0000, 0xCC01, 0xD801, 0x1400, 0xF001, 0x3C00, 0x2800, 0xE401,
	0xA001, 0x6C00, 0x7800, 0xB401, 0x5000, 0x9C01, 0x8801, 0x4400,
}

// fitCRC updates a FIT CRC with data.
func fitCRC(crc uint16, data []byte) uint16 {
	for _, b := range data {
		tmp := fitCRCTable[crc&0xF]
		crc = (crc >> 4) & 0x0FFF
		crc = crc ^ tmp ^ fitCRCTable[b&0xF]
		tmp = fitCRCTable[crc&0xF]
		crc = (crc >> 4) & 0x0FFF
		crc = crc ^ tmp ^ fitCRCTable[(b>>4)&0xF]
	}
	return crc
}

// ElapsedSeconds estimates the elapsed seconds at each point of a track:
// distance at a speed on the flat, plus elevation gain at a climbing rate if
// it is set, like Naismith's rule. Elevations are smoothed first, so noise
// doesn't count as climbing.
func ElapsedSeconds(points []Point, speedKMH, climbMetersPerHour float64) []float64 {
	dists := CumulativeDistances(points)
	smoothed := SmoothElevations(points)
	elapsed := make([]float64, len(points))
	speedMPS := speedKMH * 1000 / 3600
	for i := 1; i < len(points); i++ {
		dt := (dists[i] - dists[i-1]) / speedMPS
		if climb := smoothed[i] - smoothed[i-1]; climbMetersPerHour > 0 && climb > 0 {
			dt += climb / climbMetersPerHour * 3600
		}
		elapsed[i] = elapsed[i-1] + dt
	}
	return elapsed
}

// elevationSmoothingWindow is the number of points elevations are averaged
// over.
const elevationSmoothingWindow = 5

// SmoothElevations returns the elevations averaged over a small window,
// ignoring unknown elevations. It is NaN where all are unknown.
func SmoothElevations(points []Point) []float64 {
	smoothed := make([]float64, len(points))
	for i := range points {
		// Shrink the window near the ends so the start and end elevations
		// are kept.
		half := min(elevationSmoothingWindow/2, i, len(points)-1-i)
		var sum float64
		var n int
		for j := i - half; j <= i+half; j++ {
			if !math.IsNaN(points[j].Ele) {
				sum += points[j].Ele
				n++
			}
		}
		smoothed[i] = math.NaN()
		if n > 0 {
			smoothed[i] = sum / float64(n)
		}
	}
	return smoothed
}
