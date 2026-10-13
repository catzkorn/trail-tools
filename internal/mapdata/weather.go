package mapdata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/catzkorn/trail-tools/internal/geo"
)

// MaxForecastDays is how far ahead forecasts are available.
const MaxForecastDays = 16

// maxForecastLocations is the maximum number of locations per request.
const maxForecastLocations = 50

// Weather is the forecast weather for an hour.
type Weather struct {
	Time time.Time
	// The UTC offset of the local time zone at the location.
	UTCOffset                time.Duration
	TemperatureCelsius       float64
	ApparentTemperature      float64
	PrecipitationProbability int
	PrecipitationMM          float64
	SnowfallCM               float64
	WeatherCode              int
	CloudCoverPercent        int
	WindSpeedKMH             float64
	WindGustsKMH             float64
	WindDirectionDegrees     int
	FreezingLevelMeters      float64
	UVIndex                  float64
	IsDay                    bool
}

// ForecastQuery is a location and time to forecast.
type ForecastQuery struct {
	Point geo.Point
	Time  time.Time
}

const forecastHourlyVariables = "temperature_2m,apparent_temperature,precipitation_probability," +
	"precipitation,snowfall,weather_code,cloud_cover,wind_speed_10m,wind_gusts_10m," +
	"wind_direction_10m,freezing_level_height,uv_index,is_day"

type forecastResponse struct {
	UTCOffsetSeconds int `json:"utc_offset_seconds"`
	Hourly           struct {
		Time                     []int64    `json:"time"`
		Temperature              []*float64 `json:"temperature_2m"`
		ApparentTemperature      []*float64 `json:"apparent_temperature"`
		PrecipitationProbability []*float64 `json:"precipitation_probability"`
		Precipitation            []*float64 `json:"precipitation"`
		Snowfall                 []*float64 `json:"snowfall"`
		WeatherCode              []*float64 `json:"weather_code"`
		CloudCover               []*float64 `json:"cloud_cover"`
		WindSpeed                []*float64 `json:"wind_speed_10m"`
		WindGusts                []*float64 `json:"wind_gusts_10m"`
		WindDirection            []*float64 `json:"wind_direction_10m"`
		FreezingLevel            []*float64 `json:"freezing_level_height"`
		UVIndex                  []*float64 `json:"uv_index"`
		IsDay                    []*float64 `json:"is_day"`
	} `json:"hourly"`
}

// Forecast returns the hourly weather forecast for each query, for the hour
// containing the query time, using Open-Meteo. The forecast is downscaled to
// the elevation of each point if it is known.
func (c *Client) Forecast(ctx context.Context, queries []ForecastQuery) ([]Weather, error) {
	weather := make([]Weather, 0, len(queries))
	for start := 0; start < len(queries); start += maxForecastLocations {
		batch := queries[start:min(start+maxForecastLocations, len(queries))]
		results, err := c.forecastBatch(ctx, batch)
		if err != nil {
			return nil, err
		}
		weather = append(weather, results...)
	}
	return weather, nil
}

func (c *Client) forecastBatch(ctx context.Context, queries []ForecastQuery) ([]Weather, error) {
	var lats, lngs, eles []string
	first, last := queries[0].Time, queries[0].Time
	for _, q := range queries {
		lats = append(lats, strconv.FormatFloat(q.Point.Lat, 'f', 5, 64))
		lngs = append(lngs, strconv.FormatFloat(q.Point.Lng, 'f', 5, 64))
		ele := "nan"
		if q.Point.Ele == q.Point.Ele {
			ele = strconv.FormatFloat(q.Point.Ele, 'f', 0, 64)
		}
		eles = append(eles, ele)
		if q.Time.Before(first) {
			first = q.Time
		}
		if q.Time.After(last) {
			last = q.Time
		}
	}
	v := url.Values{}
	v.Set("latitude", strings.Join(lats, ","))
	v.Set("longitude", strings.Join(lngs, ","))
	v.Set("elevation", strings.Join(eles, ","))
	v.Set("hourly", forecastHourlyVariables)
	// Use the local time zone of each location so the response includes its
	// UTC offset. Pad the dates by a day, as they are local dates.
	v.Set("timezone", "auto")
	v.Set("timeformat", "unixtime")
	v.Set("start_date", first.UTC().AddDate(0, 0, -1).Format(time.DateOnly))
	endDate := last.UTC().AddDate(0, 0, 1)
	if maxEnd := time.Now().UTC().AddDate(0, 0, MaxForecastDays-1); endDate.After(maxEnd) {
		endDate = maxEnd
	}
	v.Set("end_date", endDate.Format(time.DateOnly))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.conf.ForecastURL+"?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	// A single location returns an object, multiple locations an array.
	var raw json.RawMessage
	if err := c.do(req, &raw); err != nil {
		return nil, fmt.Errorf("weather forecast failed: %w", err)
	}
	var responses []forecastResponse
	if len(queries) == 1 {
		var resp forecastResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, fmt.Errorf("failed to decode forecast: %w", err)
		}
		responses = []forecastResponse{resp}
	} else if err := json.Unmarshal(raw, &responses); err != nil {
		return nil, fmt.Errorf("failed to decode forecast: %w", err)
	}
	if len(responses) != len(queries) {
		return nil, fmt.Errorf("forecast returned %d locations for %d queries", len(responses), len(queries))
	}
	weather := make([]Weather, len(queries))
	for i, q := range queries {
		hour := q.Time.UTC().Truncate(time.Hour).Unix()
		idx := slices.Index(responses[i].Hourly.Time, hour)
		if idx < 0 {
			return nil, fmt.Errorf("no forecast available for %s", q.Time.UTC().Format(time.RFC3339))
		}
		weather[i] = responses[i].hour(idx)
	}
	return weather, nil
}

// hour returns the weather of the hour at index idx.
func (r *forecastResponse) hour(idx int) Weather {
	hourly := r.Hourly
	at := func(values []*float64) float64 {
		if idx >= len(values) || values[idx] == nil {
			return 0
		}
		return *values[idx]
	}
	return Weather{
		Time:                     time.Unix(hourly.Time[idx], 0).UTC(),
		UTCOffset:                time.Duration(r.UTCOffsetSeconds) * time.Second,
		TemperatureCelsius:       at(hourly.Temperature),
		ApparentTemperature:      at(hourly.ApparentTemperature),
		PrecipitationProbability: int(at(hourly.PrecipitationProbability)),
		PrecipitationMM:          at(hourly.Precipitation),
		SnowfallCM:               at(hourly.Snowfall),
		WeatherCode:              int(at(hourly.WeatherCode)),
		CloudCoverPercent:        int(at(hourly.CloudCover)),
		WindSpeedKMH:             at(hourly.WindSpeed),
		WindGustsKMH:             at(hourly.WindGusts),
		WindDirectionDegrees:     int(at(hourly.WindDirection)),
		FreezingLevelMeters:      at(hourly.FreezingLevel),
		UVIndex:                  at(hourly.UVIndex),
		IsDay:                    at(hourly.IsDay) == 1,
	}
}

// HourlyForecast returns the hourly weather forecast at a point for the
// local dates from start to end, inclusive, using Open-Meteo. Dates past the
// end of the forecast are left out. The forecast is downscaled to the
// elevation of the point if it is known.
func (c *Client) HourlyForecast(ctx context.Context, point geo.Point, start, end time.Time) ([]Weather, error) {
	if maxEnd := time.Now().UTC().AddDate(0, 0, MaxForecastDays-1); end.After(maxEnd) {
		end = maxEnd
	}
	if end.Before(start) {
		return nil, nil
	}
	v := url.Values{}
	v.Set("latitude", strconv.FormatFloat(point.Lat, 'f', 5, 64))
	v.Set("longitude", strconv.FormatFloat(point.Lng, 'f', 5, 64))
	if point.Ele == point.Ele {
		v.Set("elevation", strconv.FormatFloat(point.Ele, 'f', 0, 64))
	}
	v.Set("hourly", forecastHourlyVariables)
	// With the local time zone the dates are local dates.
	v.Set("timezone", "auto")
	v.Set("timeformat", "unixtime")
	v.Set("start_date", start.Format(time.DateOnly))
	v.Set("end_date", end.Format(time.DateOnly))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.conf.ForecastURL+"?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var resp forecastResponse
	if err := c.do(req, &resp); err != nil {
		return nil, fmt.Errorf("weather forecast failed: %w", err)
	}
	weather := make([]Weather, len(resp.Hourly.Time))
	for i := range resp.Hourly.Time {
		weather[i] = resp.hour(i)
	}
	return weather, nil
}

// WeatherDescription describes a WMO weather code.
func WeatherDescription(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Fog"
	case 51:
		return "Light drizzle"
	case 53:
		return "Drizzle"
	case 55:
		return "Dense drizzle"
	case 56, 57:
		return "Freezing drizzle"
	case 61:
		return "Light rain"
	case 63:
		return "Rain"
	case 65:
		return "Heavy rain"
	case 66, 67:
		return "Freezing rain"
	case 71:
		return "Light snow"
	case 73:
		return "Snow"
	case 75:
		return "Heavy snow"
	case 77:
		return "Snow grains"
	case 80:
		return "Light showers"
	case 81:
		return "Showers"
	case 82:
		return "Violent showers"
	case 85:
		return "Light snow showers"
	case 86:
		return "Heavy snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with hail"
	}
	return "Unknown"
}
