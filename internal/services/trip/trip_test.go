package trip

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"math"
	"strings"
	"testing"
	"time"

	activitiesv1 "github.com/johanbrandhorst/trail-tools/internal/gen/activities/v1"
	"github.com/johanbrandhorst/trail-tools/internal/geo"
	"github.com/johanbrandhorst/trail-tools/internal/trips"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		title, want string
	}{
		{"Morning Run", "morning-run"},
		{"  Säntis via the Lisengrat!  ", "santis-via-the-lisengrat"},
		{"Grand Tour of Mont-Blanc -- day 2/7", "grand-tour-of-mont-blanc-day-2-7"},
		{"Großglockner", "grossglockner"},
		{"Ørsta & Æsir", "rsta-sir"},
		{"🏔️⛰️", ""},
		{strings.Repeat("long ", 30), strings.TrimSuffix(strings.Repeat("long-", 16), "-")},
	}
	for _, tt := range tests {
		got := Slugify(tt.title)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.title, got, tt.want)
		}
		if got != "" {
			if err := ValidateSlug(got); err != nil {
				t.Errorf("Slugify(%q) = %q, which is invalid: %v", tt.title, got, err)
			}
		}
	}
}

func TestValidateSlug(t *testing.T) {
	for _, slug := range []string{"a", "my-first-ultra", "2026-utmb"} {
		if err := ValidateSlug(slug); err != nil {
			t.Errorf("ValidateSlug(%q) = %v, want nil", slug, err)
		}
	}
	for _, slug := range []string{"", "-a", "a-", "a--b", "A", "a b", "a_b", "ä", strings.Repeat("a", 81)} {
		if err := ValidateSlug(slug); err == nil {
			t.Errorf("ValidateSlug(%q) = nil, want an error", slug)
		}
	}
}

func TestValidateHandle(t *testing.T) {
	for _, handle := range []string{"abc", "jane-doe", "runner42", "a-b", strings.Repeat("a", 30)} {
		if err := validateHandle(handle); err != nil {
			t.Errorf("validateHandle(%q) = %v, want nil", handle, err)
		}
	}
	for _, handle := range []string{"", "ab", "-abc", "abc-", "Jane", "jane doe", "jane_doe", "jané", strings.Repeat("a", 31)} {
		if err := validateHandle(handle); err == nil {
			t.Errorf("validateHandle(%q) = nil, want an error", handle)
		}
	}
}

func TestExcerpt(t *testing.T) {
	tests := []struct {
		text      string
		maxLength int
		want      string
	}{
		{"", 10, ""},
		{"Short story.", 20, "Short story."},
		{"First paragraph.\n\nSecond   paragraph.", 100, "First paragraph. Second paragraph."},
		{"The trail climbed steeply through the forest", 25, "The trail climbed…"},
		{"Supercalifragilisticexpialidocious", 10, "Supercali…"},
	}
	for _, tt := range tests {
		if got := Excerpt(tt.text, tt.maxLength); got != tt.want {
			t.Errorf("Excerpt(%q, %d) = %q, want %q", tt.text, tt.maxLength, got, tt.want)
		}
		if got := Excerpt(tt.text, tt.maxLength); len([]rune(got)) > tt.maxLength {
			t.Errorf("Excerpt(%q, %d) = %q, which is too long", tt.text, tt.maxLength, got)
		}
	}
}

// testTrack is a track heading north from the equator, with a point every
// 100 seconds and about 111 meters, and a pause of 300 seconds at the third
// point.
func testTrack() *track {
	return newTrack(activitiesv1.ActivityStreams_builder{
		TimeSeconds:    []int32{0, 100, 200, 500, 600},
		DistanceMeters: []float64{0, 111, 222, 222, 333},
		Lat:            []float64{0, 0.001, 0.002, 0.002, 0.003},
		Lng:            []float64{0, 0, 0, 0, 0},
		AltitudeMeters: []float64{100, 110, 120, 120, 130},
	}.Build())
}

func TestPositionAt(t *testing.T) {
	tr := testTrack()
	tests := []struct {
		name    string
		elapsed float64
		wantOK  bool
		wantLat float64
		dist    float64
	}{
		{name: "start", elapsed: 0, wantOK: true, wantLat: 0, dist: 0},
		{name: "between points", elapsed: 150, wantOK: true, wantLat: 0.0015, dist: 166.5},
		{name: "during a pause", elapsed: 350, wantOK: true, wantLat: 0.002, dist: 222},
		{name: "end", elapsed: 600, wantOK: true, wantLat: 0.003, dist: 333},
		{name: "shortly before the start", elapsed: -300, wantOK: true, wantLat: 0, dist: 0},
		{name: "shortly after the end", elapsed: 1000, wantOK: true, wantLat: 0.003, dist: 333},
		{name: "long before the start", elapsed: -3600, wantOK: false},
		{name: "long after the end", elapsed: 4000, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, dist, ok := tr.positionAt(tt.elapsed)
			if ok != tt.wantOK {
				t.Fatalf("positionAt(%v) ok = %v, want %v", tt.elapsed, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if math.Abs(p.Lat-tt.wantLat) > 1e-9 || p.Lng != 0 {
				t.Errorf("positionAt(%v) = %v, want latitude %v", tt.elapsed, p, tt.wantLat)
			}
			if math.Abs(dist-tt.dist) > 1e-6 {
				t.Errorf("positionAt(%v) distance = %v, want %v", tt.elapsed, dist, tt.dist)
			}
		})
	}

	// Tracks without times, e.g. from a summary polyline, can't place
	// photos by time.
	noTimes := trackFromPolyline(geo.EncodePolyline([]geo.Point{{Lat: 0, Lng: 0}, {Lat: 0.01, Lng: 0}}))
	if _, _, ok := newTrack(noTimes).positionAt(0); ok {
		t.Error("positionAt on a track without times succeeded")
	}
	var missing *track
	if _, _, ok := missing.positionAt(0); ok {
		t.Error("positionAt on a missing track succeeded")
	}
}

func TestNearestDistance(t *testing.T) {
	tr := testTrack()
	dist, ok := tr.nearestDistance(geo.Point{Lat: 0.0029, Lng: 0.0001})
	if !ok || dist != 333 {
		t.Errorf("nearestDistance = %v, %v, want 333, true", dist, ok)
	}
	var missing *track
	if _, ok := missing.nearestDistance(geo.Point{}); ok {
		t.Error("nearestDistance on a missing track succeeded")
	}
}

func TestTrackFromStreams(t *testing.T) {
	var streams activitiesv1.ActivityStreams
	for i := range 5000 {
		streams.SetTimeSeconds(append(streams.GetTimeSeconds(), int32(i)))
		streams.SetDistanceMeters(append(streams.GetDistanceMeters(), float64(i)))
		streams.SetLat(append(streams.GetLat(), float64(i)*1e-5))
		streams.SetLng(append(streams.GetLng(), 0))
		streams.SetHeartrateBpm(append(streams.GetHeartrateBpm(), 150))
	}
	got := trackFromStreams(&streams)
	if n := len(got.GetLat()); n > maxTrackPoints || n < maxTrackPoints/2 {
		t.Errorf("track has %d points, want at most %d", n, maxTrackPoints)
	}
	if len(got.GetTimeSeconds()) != len(got.GetLat()) || len(got.GetDistanceMeters()) != len(got.GetLat()) {
		t.Error("track lost its times or distances")
	}
	if len(got.GetHeartrateBpm()) != 0 || len(got.GetAltitudeMeters()) != 0 {
		t.Error("track has streams it shouldn't")
	}
	if trackFromStreams(&activitiesv1.ActivityStreams{}) != nil {
		t.Error("track from empty streams is not nil")
	}
}

func TestRoutePoints(t *testing.T) {
	points := newTrack(activitiesv1.ActivityStreams_builder{
		Lat: []float64{0, 0.001},
		Lng: []float64{0, 0},
	}.Build()).routePoints()
	if len(points) != 2 || points[1].GetElevationMeters() != 0 || math.Abs(points[1].GetDistanceMeters()-111.2) > 0.1 {
		t.Errorf("unexpected route points %v", points)
	}
}

func TestParseLocalTime(t *testing.T) {
	got, err := parseLocalTime("2026-09-07T10:15:30", "Europe/Zurich")
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 9, 7, 8, 15, 30, 0, time.UTC); !got.Equal(want) {
		t.Errorf("parseLocalTime = %v, want %v", got, want)
	}
	got, err = parseLocalTime("2026-09-07T10:15:30", "Not/AZone")
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 9, 7, 10, 15, 30, 0, time.UTC); !got.Equal(want) {
		t.Errorf("parseLocalTime with an unknown zone = %v, want %v", got, want)
	}
	if _, err := parseLocalTime("2026:09:07 10:15:30", "UTC"); err == nil {
		t.Error("parseLocalTime of an EXIF formatted time succeeded")
	}
}

func encodeTestImage(t *testing.T, w, h int, asPNG bool) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: 100, A: 255})
		}
	}
	// A transparent corner, which must become white rather than black.
	img.Set(0, 0, color.NRGBA{})
	var buf bytes.Buffer
	var err error
	if asPNG {
		err = png.Encode(&buf, img)
	} else {
		err = jpeg.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestProcessImage(t *testing.T) {
	tests := []struct {
		name                string
		w, h                int
		png                 bool
		wantW, wantH        int
		wantThumbW, wantThW int
	}{
		{name: "large landscape JPEG", w: 4000, h: 3000, wantW: 2048, wantH: 1536, wantThumbW: 480, wantThW: 360},
		{name: "portrait PNG", w: 600, h: 1200, png: true, wantW: 600, wantH: 1200, wantThumbW: 240, wantThW: 480},
		{name: "small", w: 300, h: 200, wantW: 300, wantH: 200, wantThumbW: 300, wantThW: 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processImage(encodeTestImage(t, tt.w, tt.h, tt.png))
			if err != nil {
				t.Fatal(err)
			}
			if got.width != tt.wantW || got.height != tt.wantH {
				t.Errorf("large is %dx%d, want %dx%d", got.width, got.height, tt.wantW, tt.wantH)
			}
			large, err := jpeg.Decode(bytes.NewReader(got.large))
			if err != nil {
				t.Fatalf("large is not a JPEG: %v", err)
			}
			if b := large.Bounds(); b.Dx() != tt.wantW || b.Dy() != tt.wantH {
				t.Errorf("decoded large is %v", b)
			}
			thumb, err := jpeg.Decode(bytes.NewReader(got.thumb))
			if err != nil {
				t.Fatalf("thumbnail is not a JPEG: %v", err)
			}
			if b := thumb.Bounds(); b.Dx() != tt.wantThumbW || b.Dy() != tt.wantThW {
				t.Errorf("thumbnail is %dx%d, want %dx%d", b.Dx(), b.Dy(), tt.wantThumbW, tt.wantThW)
			}
			if tt.png {
				if r, g, b, _ := large.At(0, 0).RGBA(); r>>8 < 200 || g>>8 < 200 || b>>8 < 200 {
					t.Errorf("transparent pixel is %v, want white", large.At(0, 0))
				}
			}
		})
	}
	for _, data := range [][]byte{[]byte("not an image"), []byte("GIF89a")} {
		if _, err := processImage(data); err == nil {
			t.Errorf("processImage(%q) succeeded", data)
		}
	}
}

func TestSummarize(t *testing.T) {
	r := &trips.Report{
		SportType:           "TrailRun",
		DistanceMeters:      21097,
		ElevationGainMeters: 1234.4,
		Date:                time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC),
	}
	profile := &trips.Profile{DisplayName: "Jane Doe"}
	want := "A 21.1 km trail run with 1,234 m of climbing by Jane Doe, on 7 September 2026."
	if got := summarize(r, profile); got != want {
		t.Errorf("summarize = %q, want %q", got, want)
	}
	if got, want := summarize(&trips.Report{}, profile), "A trip by Jane Doe."; got != want {
		t.Errorf("summarize = %q, want %q", got, want)
	}
}

func TestThousands(t *testing.T) {
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 1234567: "1,234,567", -1234: "-1,234"} {
		if got := thousands(n); got != want {
			t.Errorf("thousands(%d) = %q, want %q", n, got, want)
		}
	}
}
