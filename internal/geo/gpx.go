package geo

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
)

type gpxFile struct {
	XMLName xml.Name   `xml:"gpx"`
	Name    string     `xml:"metadata>name"`
	Tracks  []gpxTrack `xml:"trk"`
	Routes  []gpxRoute `xml:"rte"`
}

type gpxTrack struct {
	Name     string       `xml:"name"`
	Segments []gpxSegment `xml:"trkseg"`
}

type gpxSegment struct {
	Points []gpxPoint `xml:"trkpt"`
}

type gpxRoute struct {
	Name   string     `xml:"name"`
	Points []gpxPoint `xml:"rtept"`
}

type gpxPoint struct {
	Lat float64  `xml:"lat,attr"`
	Lng float64  `xml:"lon,attr"`
	Ele *float64 `xml:"ele"`
}

// GPXTrack is a track read from a GPX file.
type GPXTrack struct {
	// The name of the track, or the file if the track has no name.
	Name string
	// The points of the track. Elevations are NaN if the file has none.
	Points []Point
}

// ParseGPX reads the first track (or route, if there are no tracks) from a GPX
// file. All segments of the track are joined.
func ParseGPX(r io.Reader) (*GPXTrack, error) {
	var f gpxFile
	if err := xml.NewDecoder(r).Decode(&f); err != nil {
		return nil, fmt.Errorf("failed to parse GPX: %w", err)
	}
	track := &GPXTrack{Name: f.Name}
	var gpxPoints []gpxPoint
	switch {
	case len(f.Tracks) > 0:
		if f.Tracks[0].Name != "" {
			track.Name = f.Tracks[0].Name
		}
		for _, seg := range f.Tracks[0].Segments {
			gpxPoints = append(gpxPoints, seg.Points...)
		}
	case len(f.Routes) > 0:
		if f.Routes[0].Name != "" {
			track.Name = f.Routes[0].Name
		}
		gpxPoints = f.Routes[0].Points
	}
	if len(gpxPoints) < 2 {
		return nil, errors.New("GPX file must contain a track or route with at least 2 points")
	}
	for _, p := range gpxPoints {
		if p.Lat < -90 || p.Lat > 90 || p.Lng < -180 || p.Lng > 180 {
			return nil, fmt.Errorf("invalid GPX coordinate %v,%v", p.Lat, p.Lng)
		}
		ele := math.NaN()
		if p.Ele != nil {
			ele = *p.Ele
		}
		track.Points = append(track.Points, Point{Lat: p.Lat, Lng: p.Lng, Ele: ele})
	}
	return track, nil
}

// WriteGPX encodes the points as a GPX 1.1 file with a single track.
func WriteGPX(w io.Writer, name string, points []Point) error {
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(`<gpx version="1.1" creator="trailrunn.ing" xmlns="http://www.topografix.com/GPX/1/1">` + "\n")
	buf.WriteString("  <metadata><name>")
	xml.EscapeText(&buf, []byte(name))
	buf.WriteString("</name></metadata>\n  <trk>\n    <name>")
	xml.EscapeText(&buf, []byte(name))
	buf.WriteString("</name>\n    <trkseg>\n")
	for _, p := range points {
		buf.WriteString(`      <trkpt lat="`)
		buf.WriteString(strconv.FormatFloat(p.Lat, 'f', 6, 64))
		buf.WriteString(`" lon="`)
		buf.WriteString(strconv.FormatFloat(p.Lng, 'f', 6, 64))
		buf.WriteString(`">`)
		if !math.IsNaN(p.Ele) {
			buf.WriteString("<ele>")
			buf.WriteString(strconv.FormatFloat(p.Ele, 'f', 1, 64))
			buf.WriteString("</ele>")
		}
		buf.WriteString("</trkpt>\n")
	}
	buf.WriteString("    </trkseg>\n  </trk>\n</gpx>\n")
	_, err := w.Write(buf.Bytes())
	return err
}
