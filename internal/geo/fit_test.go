package geo

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"
	"time"
)

// fitMessage is a decoded FIT data message: its global number and the raw
// bytes of each field by field number.
type fitMessage struct {
	global uint16
	fields map[uint8][]byte
}

func (m fitMessage) uint(t *testing.T, num uint8) uint64 {
	t.Helper()
	b, ok := m.fields[num]
	if !ok {
		t.Fatalf("message %d has no field %d", m.global, num)
	}
	var v uint64
	for i := len(b) - 1; i >= 0; i-- {
		v = v<<8 | uint64(b[i])
	}
	return v
}

func (m fitMessage) sint32(t *testing.T, num uint8) int32 {
	t.Helper()
	return int32(uint32(m.uint(t, num)))
}

func (m fitMessage) string(num uint8) string {
	return string(bytes.TrimRight(m.fields[num], "\x00"))
}

// decodeFIT decodes a FIT file with little endian definitions and normal
// headers, checking its header and CRCs.
func decodeFIT(t *testing.T, file []byte) []fitMessage {
	t.Helper()
	if len(file) < 16 || file[0] != 14 || string(file[8:12]) != ".FIT" {
		t.Fatalf("invalid header % x", file[:min(len(file), 14)])
	}
	if crc := binary.LittleEndian.Uint16(file[12:]); crc != fitCRC(0, file[:12]) {
		t.Fatalf("header CRC %x, want %x", crc, fitCRC(0, file[:12]))
	}
	size := int(binary.LittleEndian.Uint32(file[4:]))
	if len(file) != 14+size+2 {
		t.Fatalf("file is %d bytes, header says %d of data", len(file), size)
	}
	if crc := binary.LittleEndian.Uint16(file[14+size:]); crc != fitCRC(0, file[:14+size]) {
		t.Fatalf("file CRC %x, want %x", crc, fitCRC(0, file[:14+size]))
	}
	type definition struct {
		global uint16
		fields []fitField
	}
	defs := map[uint8]definition{}
	var messages []fitMessage
	data := file[14 : 14+size]
	for len(data) > 0 {
		header := data[0]
		local := header & 0x0F
		data = data[1:]
		if header&0x40 != 0 {
			def := definition{global: binary.LittleEndian.Uint16(data[2:])}
			n := int(data[4])
			data = data[5:]
			for i := range n {
				def.fields = append(def.fields, fitField{data[i*3], data[i*3+1], data[i*3+2]})
			}
			data = data[n*3:]
			defs[local] = def
			continue
		}
		def, ok := defs[local]
		if !ok {
			t.Fatalf("data message of undefined local type %d", local)
		}
		m := fitMessage{global: def.global, fields: map[uint8][]byte{}}
		for _, field := range def.fields {
			m.fields[field.num] = data[:field.size]
			data = data[field.size:]
		}
		messages = append(messages, m)
	}
	return messages
}

func TestWriteFITCourse(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 3, 7, 0, 0, 0, time.UTC)
	points := []Point{
		{Lat: 49.7, Lng: -123.15, Ele: 10},
		{Lat: 49.71, Lng: -123.15, Ele: 510},
		{Lat: 49.72, Lng: -123.15, Ele: math.NaN()},
	}
	var buf bytes.Buffer
	err := WriteFITCourse(&buf, &FITCourse{
		Name:           "Squamish 50K",
		Points:         points,
		ElapsedSeconds: []float64{0, 1800, 3000},
		CoursePoints: []CoursePoint{
			{Name: "Alice Lake aid station", DistanceMeters: 1000, ElapsedSeconds: 1500, Type: CoursePointFood},
			{Name: "Finish", DistanceMeters: 99999, ElapsedSeconds: 3000},
		},
		Start: start,
	})
	if err != nil {
		t.Fatalf("WriteFITCourse: %v", err)
	}
	byGlobal := map[uint16][]fitMessage{}
	for _, m := range decodeFIT(t, buf.Bytes()) {
		byGlobal[m.global] = append(byGlobal[m.global], m)
	}
	fitStart := uint64(start.Sub(fitEpoch) / time.Second)

	if fileID := byGlobal[fitFileID]; len(fileID) != 1 || fileID[0].uint(t, 0) != 6 {
		t.Errorf("file ID %v, want a course file", fileID)
	}
	if course := byGlobal[fitCourse]; len(course) != 1 || course[0].string(5) != "Squamish 50K" || course[0].uint(t, 4) != 1 {
		t.Errorf("course %v, want a running course named Squamish 50K", course)
	}
	records := byGlobal[fitRecord]
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}
	dists := CumulativeDistances(points)
	for i, r := range records {
		if got := r.uint(t, 253); got != fitStart+[]uint64{0, 1800, 3000}[i] {
			t.Errorf("record %d at %d, want %d", i, got, fitStart)
		}
		if got := float64(r.sint32(t, 0)) * 180 / (1 << 31); math.Abs(got-points[i].Lat) > 1e-6 {
			t.Errorf("record %d latitude %v, want %v", i, got, points[i].Lat)
		}
		if got := float64(r.uint(t, 5)) / 100; math.Abs(got-dists[i]) > 0.01 {
			t.Errorf("record %d distance %v, want %v", i, got, dists[i])
		}
	}
	if alt := records[1].uint(t, 2); alt != (510+500)*5 {
		t.Errorf("altitude %d, want %d", alt, (510+500)*5)
	}
	if alt := records[2].uint(t, 2); alt != fitInvalidUint16 {
		t.Errorf("unknown altitude %d, want invalid", alt)
	}
	laps := byGlobal[fitLap]
	if len(laps) != 1 || laps[0].uint(t, 7) != 3000*1000 || laps[0].uint(t, 21) != 500 {
		t.Errorf("lap %v, want 3000s and 500m up", laps)
	}
	events := byGlobal[fitEvent]
	if len(events) != 2 || events[0].uint(t, 1) != 0 || events[1].uint(t, 1) != 9 || events[1].uint(t, 253) != fitStart+3000 {
		t.Errorf("events %v, want the timer to start and stop", events)
	}

	// Course points are placed along the course, with names cut to fit.
	cps := byGlobal[fitCoursePoint]
	if len(cps) != 2 {
		t.Fatalf("got %d course points, want 2", len(cps))
	}
	if name := cps[0].string(6); name != "Alice Lake aid " || cps[0].uint(t, 5) != uint64(CoursePointFood) {
		t.Errorf("course point %q type %d", name, cps[0].uint(t, 5))
	}
	if d := cps[0].uint(t, 4); d != 100000 {
		t.Errorf("course point at %d cm, want 100000", d)
	}
	if at := cps[0].uint(t, 1) - fitStart; at != 1500 {
		t.Errorf("course point at %d s, want 1500", at)
	}
	if lat := float64(cps[0].sint32(t, 2)) * 180 / (1 << 31); math.Abs(lat-(49.7+0.01*1000/dists[1])) > 1e-6 {
		t.Errorf("course point latitude %v", lat)
	}
	// Course points past the end are at the finish.
	if d, name := float64(cps[1].uint(t, 4))/100, cps[1].string(6); math.Abs(d-dists[2]) > 0.01 || name != "Finish" {
		t.Errorf("finish course point %q at %v, want %v", name, d, dists[2])
	}

	for name, c := range map[string]*FITCourse{
		"one point":     {Points: points[:1], ElapsedSeconds: []float64{0}},
		"missing times": {Points: points, ElapsedSeconds: []float64{0}},
	} {
		if err := WriteFITCourse(&buf, c); err == nil {
			t.Errorf("WriteFITCourse with %s succeeded", name)
		}
	}
}

func TestFITStringValue(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		in   string
		size int
		want string
	}{
		{"Finish", 16, "Finish"},
		{"Col de la Seigne", 16, "Col de la Seign"},
		// Multibyte characters aren't cut in half.
		{strings.Repeat("é", 10), 16, strings.Repeat("é", 7)},
	} {
		got := fitStringValue(tc.in, tc.size)
		if len(got) != tc.size || string(bytes.TrimRight(got, "\x00")) != tc.want {
			t.Errorf("fitStringValue(%q, %d) = %q, want %q", tc.in, tc.size, got, tc.want)
		}
	}
}

func TestElapsedSeconds(t *testing.T) {
	t.Parallel()
	// 1 km north climbing 100 m at 10 km/h and 600 m/h takes 6 + 10
	// minutes; unknown elevations add no climbing.
	points := []Point{{Lat: 0, Ele: 0}, {Lat: 0.008993, Ele: 100}}
	elapsed := ElapsedSeconds(points, 10, 600)
	if math.Abs(elapsed[1]-960) > 1 {
		t.Errorf("got %v seconds, want 960", elapsed[1])
	}
	unknown := []Point{{Lat: 0, Ele: math.NaN()}, {Lat: 0.008993, Ele: 100}}
	if elapsed := ElapsedSeconds(unknown, 10, 600); math.Abs(elapsed[1]-360) > 1 {
		t.Errorf("got %v seconds with an unknown elevation, want 360", elapsed[1])
	}
}
