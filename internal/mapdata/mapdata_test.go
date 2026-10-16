package mapdata_test

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/johanbrandhorst/trail-tools/internal/geo"
	"github.com/johanbrandhorst/trail-tools/internal/mapdata"
)

// fakeElevation returns elevations 1000 times the latitude.
type fakeElevation struct {
	lookups int
}

func (f *fakeElevation) Points(_ context.Context, points []geo.Point) ([]float64, error) {
	f.lookups += len(points)
	elevations := make([]float64, len(points))
	for i, p := range points {
		elevations[i] = p.Lat * 1000
	}
	return elevations, nil
}

func TestFillElevations(t *testing.T) {
	t.Parallel()
	elevation := &fakeElevation{}
	client := mapdata.New(mapdata.Config{Elevation: elevation, HTTPClient: &http.Client{}})
	var points []geo.Point
	for i := range 250 {
		points = append(points, geo.Point{Lat: float64(i) * 0.001})
	}
	if err := client.FillElevations(context.Background(), points, 150); err != nil {
		t.Fatalf("FillElevations: %v", err)
	}
	if elevation.lookups > 150 {
		t.Errorf("looked up %d points, want at most 150", elevation.lookups)
	}
	for i, p := range points {
		if diff := p.Ele - p.Lat*1000; diff > 0.01 || diff < -0.01 {
			t.Fatalf("point %d: got elevation %v, want %v", i, p.Ele, p.Lat*1000)
		}
	}
}

func TestStatusErrorPlainText(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGatewayTimeout)
		fmt.Fprint(w, `<?xml version="1.0"?><html><head><title>Nominatim Response</title></head>
<body><p>The data is from OpenStreetMap.</p>
<p><strong>Error</strong>: runtime error: the server is probably too busy.</p></body></html>`)
	}))
	t.Cleanup(srv.Close)
	client := mapdata.New(mapdata.Config{NominatimURL: srv.URL, HTTPClient: srv.Client()})
	_, err := client.Search(context.Background(), "Squamish", 5)
	if err == nil || strings.Contains(err.Error(), "<") || !strings.Contains(err.Error(), "Error : runtime error: the server is probably too busy.") {
		t.Errorf("unexpected error %v", err)
	}
}

func TestHourlyForecast(t *testing.T) {
	t.Parallel()
	start := time.Now().UTC().Truncate(24 * time.Hour)
	var query url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		first := start.Add(-7 * time.Hour).Unix()
		fmt.Fprintf(w, `{"utc_offset_seconds":-25200,"hourly":{"time":[%d,%d],"temperature_2m":[4.5,null],"weather_code":[61,3],"is_day":[0,1]}}`, first, first+3600)
	}))
	t.Cleanup(srv.Close)
	client := mapdata.New(mapdata.Config{ForecastURL: srv.URL, HTTPClient: srv.Client(), Elevation: &fakeElevation{}})
	weather, err := client.HourlyForecast(context.Background(), geo.Point{Lat: 49.7, Lng: -123.15, Ele: math.NaN()}, start, start.AddDate(0, 1, 0))
	if err != nil {
		t.Fatalf("HourlyForecast: %v", err)
	}
	// The dates are local dates, and the end is the end of the forecast.
	wantEnd := time.Now().UTC().AddDate(0, 0, mapdata.MaxForecastDays-1).Format(time.DateOnly)
	if query.Get("start_date") != start.Format(time.DateOnly) || query.Get("end_date") != wantEnd ||
		query.Get("timezone") != "auto" || query.Has("elevation") || query.Get("latitude") != "49.70000" {
		t.Errorf("unexpected query %v", query)
	}
	if len(weather) != 2 || weather[0].TemperatureCelsius != 4.5 || weather[0].WeatherCode != 61 || weather[0].IsDay ||
		weather[0].UTCOffset != -7*time.Hour || !weather[1].IsDay || weather[1].TemperatureCelsius != 0 ||
		!weather[1].Time.Equal(start.Add(-6*time.Hour)) {
		t.Errorf("unexpected weather %+v", weather)
	}
	// Dates past the end of the forecast aren't requested.
	query = nil
	weather, err = client.HourlyForecast(context.Background(), geo.Point{}, start.AddDate(0, 1, 0), start.AddDate(0, 2, 0))
	if err != nil || weather != nil || query != nil {
		t.Errorf("forecast past the end: %v, %v, query %v", weather, err, query)
	}
}
