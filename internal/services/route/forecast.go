package route

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/authn"
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/mapdata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultForecastInterval = time.Hour
	minForecastInterval     = 15 * time.Minute
	maxForecastPoints       = 200
	maxForecastPathPoints   = 1000
	// elevationSmoothingWindow is the number of points averaged when
	// estimating climbing time, to filter elevation noise.
	elevationSmoothingWindow = 5
)

// forecastSample is a point along the route at an expected time.
type forecastSample struct {
	kind     routesv1.ForecastPointKind
	elapsed  time.Duration
	distance float64
	point    geo.Point
}

func (s *Service) GetRouteForecast(ctx context.Context, req *connect.Request[routesv1.GetRouteForecastRequest]) (*connect.Response[routesv1.GetRouteForecastResponse], error) {
	if _, ok := authn.GetUserID(ctx); !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	msg := req.Msg
	if !msg.HasStartTime() {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("start_time is required"))
	}
	speed := msg.GetSpeedKmh()
	if speed <= 0 || speed > 40 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("speed_kmh must be between 0 and 40"))
	}
	if msg.GetClimbMetersPerHour() < 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("climb_meters_per_hour must not be negative"))
	}
	interval := defaultForecastInterval
	if msg.GetIntervalMinutes() != 0 {
		interval = time.Duration(msg.GetIntervalMinutes()) * time.Minute
		if interval < minForecastInterval {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("interval_minutes must be at least %d", int(minForecastInterval.Minutes())))
		}
	}

	points, err := s.forecastPath(ctx, msg)
	if err != nil {
		return nil, err
	}

	start := msg.GetStartTime().AsTime()
	samples, duration := sampleRoute(points, speed, msg.GetClimbMetersPerHour(), interval)
	now := time.Now()
	if start.Before(now.Add(-time.Hour)) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("start_time must not be in the past"))
	}
	if finish := start.Add(duration); finish.After(now.AddDate(0, 0, mapdata.MaxForecastDays-1)) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(
			"forecasts are only available %d days ahead, but the route would finish at %s",
			mapdata.MaxForecastDays-1, finish.UTC().Format(time.RFC3339)))
	}
	if len(samples) > maxForecastPoints {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(
			"the route would take %s, which is too many forecast points; use a longer interval_minutes", duration.Round(time.Minute)))
	}

	queries := make([]mapdata.ForecastQuery, len(samples))
	for i, sample := range samples {
		queries[i] = mapdata.ForecastQuery{Point: sample.point, Time: start.Add(sample.elapsed)}
	}
	weather, err := s.mapData.Forecast(ctx, queries)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}

	forecastPoints := make([]*routesv1.ForecastPoint, len(samples))
	for i, sample := range samples {
		forecastPoints[i] = forecastPointToProto(sample, queries[i].Time, weather[i])
	}
	stats := geo.ComputeStats(points, nil)
	return connect.NewResponse(routesv1.GetRouteForecastResponse_builder{
		Points:              forecastPoints,
		FinishTime:          timestamppb.New(start.Add(duration)),
		DurationSeconds:     int32(duration.Seconds()),
		DistanceMeters:      stats.DistanceMeters,
		ElevationGainMeters: stats.ElevationGainMeters,
		Warnings:            forecastWarnings(samples, weather),
		Path:                PathToProto(points, maxForecastPathPoints),
	}.Build()), nil
}

// forecastPath returns the path to forecast, with elevations.
func (s *Service) forecastPath(ctx context.Context, msg *routesv1.GetRouteForecastRequest) ([]geo.Point, error) {
	var points []geo.Point
	switch msg.WhichSource() {
	case routesv1.GetRouteForecastRequest_RouteId_case:
		route, err := s.getRoute(ctx, msg.GetRouteId())
		if err != nil {
			return nil, err
		}
		points, _ = route.Path()
	case routesv1.GetRouteForecastRequest_Gpx_case:
		track, err := geo.ParseGPX(strings.NewReader(msg.GetGpx()))
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		points = track.Points
	case routesv1.GetRouteForecastRequest_Path_case:
		for _, p := range msg.GetPath().GetPoints() {
			points = append(points, geo.Point{Lat: p.GetLat(), Lng: p.GetLng(), Ele: math.NaN()})
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("one of route_id, gpx or path must be set"))
	}
	if len(points) < 2 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the route must have at least 2 points"))
	}
	if err := s.fillMissingElevations(ctx, points); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return points, nil
}

// sampleRoute estimates the elapsed time at each point of the route and
// returns samples at the start, every interval, at the highest point and at
// the finish, in time order. Time is distance at the flat speed, plus
// elevation gain at the climbing rate if it is set.
func sampleRoute(points []geo.Point, speedKMH, climbMetersPerHour float64, interval time.Duration) ([]forecastSample, time.Duration) {
	dists := geo.CumulativeDistances(points)
	smoothed := smoothElevations(points)
	elapsed := make([]float64, len(points))
	speedMPS := speedKMH * 1000 / 3600
	for i := 1; i < len(points); i++ {
		dt := (dists[i] - dists[i-1]) / speedMPS
		if climb := smoothed[i] - smoothed[i-1]; climbMetersPerHour > 0 && climb > 0 {
			dt += climb / climbMetersPerHour * 3600
		}
		elapsed[i] = elapsed[i-1] + dt
	}
	total := elapsed[len(elapsed)-1]
	at := func(kind routesv1.ForecastPointKind, idx int) forecastSample {
		return forecastSample{
			kind:     kind,
			elapsed:  time.Duration(elapsed[idx] * float64(time.Second)),
			distance: dists[idx],
			point:    points[idx],
		}
	}

	samples := []forecastSample{at(routesv1.ForecastPointKind_FORECAST_POINT_KIND_START, 0)}
	idx := 0
	for t := interval.Seconds(); t < total; t += interval.Seconds() {
		for idx < len(elapsed)-1 && elapsed[idx+1] <= t {
			idx++
		}
		// Interpolate the position between the two surrounding points.
		sample := at(routesv1.ForecastPointKind_FORECAST_POINT_KIND_INTERVAL, idx)
		if next := idx + 1; next < len(points) && elapsed[next] > elapsed[idx] {
			f := (t - elapsed[idx]) / (elapsed[next] - elapsed[idx])
			sample.point = geo.Interpolate(points[idx], points[next], f)
			sample.distance = dists[idx] + (dists[next]-dists[idx])*f
		}
		sample.elapsed = time.Duration(t * float64(time.Second))
		samples = append(samples, sample)
		if len(samples) > maxForecastPoints {
			// The caller rejects routes with too many points.
			break
		}
	}
	high := 0
	for i, p := range points {
		if p.Ele > points[high].Ele {
			high = i
		}
	}
	if high != 0 && high != len(points)-1 {
		samples = append(samples, at(routesv1.ForecastPointKind_FORECAST_POINT_KIND_HIGH_POINT, high))
	}
	samples = append(samples, at(routesv1.ForecastPointKind_FORECAST_POINT_KIND_FINISH, len(points)-1))
	sort.SliceStable(samples, func(i, j int) bool { return samples[i].elapsed < samples[j].elapsed })
	return samples, time.Duration(total * float64(time.Second))
}

// smoothElevations returns the elevations averaged over a small window.
func smoothElevations(points []geo.Point) []float64 {
	smoothed := make([]float64, len(points))
	for i := range points {
		// Shrink the window near the ends so the start and end elevations
		// are kept.
		half := min(elevationSmoothingWindow/2, i, len(points)-1-i)
		var sum float64
		var n int
		for j := i - half; j <= i+half; j++ {
			sum += points[j].Ele
			n++
		}
		smoothed[i] = sum / float64(n)
	}
	return smoothed
}

func forecastPointToProto(sample forecastSample, t time.Time, w mapdata.Weather) *routesv1.ForecastPoint {
	return routesv1.ForecastPoint_builder{
		Kind:                            sample.kind,
		Time:                            timestamppb.New(t),
		DistanceMeters:                  sample.distance,
		Location:                        geov1.LatLng_builder{Lat: sample.point.Lat, Lng: sample.point.Lng}.Build(),
		ElevationMeters:                 sample.point.Ele,
		TemperatureCelsius:              w.TemperatureCelsius,
		ApparentTemperatureCelsius:      w.ApparentTemperature,
		PrecipitationProbabilityPercent: int32(w.PrecipitationProbability),
		PrecipitationMm:                 w.PrecipitationMM,
		SnowfallCm:                      w.SnowfallCM,
		WindSpeedKmh:                    w.WindSpeedKMH,
		WindGustsKmh:                    w.WindGustsKMH,
		WindDirectionDegrees:            int32(w.WindDirectionDegrees),
		CloudCoverPercent:               int32(w.CloudCoverPercent),
		FreezingLevelMeters:             w.FreezingLevelMeters,
		UvIndex:                         w.UVIndex,
		IsDay:                           w.IsDay,
		WeatherCode:                     int32(w.WeatherCode),
		WeatherDescription:              mapdata.WeatherDescription(w.WeatherCode),
	}.Build()
}

// forecastHazard is a kind of hazard found in a forecast.
type forecastHazard struct {
	title string
	// detail returns a description of the hazard at a point, or an empty
	// string if the hazard doesn't apply.
	detail func(sample forecastSample, w mapdata.Weather) string
}

var forecastHazards = []forecastHazard{
	{
		title: "Thunderstorms possible",
		detail: func(_ forecastSample, w mapdata.Weather) string {
			if w.WeatherCode >= 95 {
				return mapdata.WeatherDescription(w.WeatherCode)
			}
			return ""
		},
	},
	{
		title: "Above the freezing level",
		detail: func(s forecastSample, w mapdata.Weather) string {
			if w.FreezingLevelMeters > 0 && s.point.Ele > w.FreezingLevelMeters {
				return fmt.Sprintf("at %.0fm, freezing level %.0fm", s.point.Ele, w.FreezingLevelMeters)
			}
			return ""
		},
	},
	{
		title: "Strong wind gusts",
		detail: func(_ forecastSample, w mapdata.Weather) string {
			if w.WindGustsKMH >= 60 {
				return fmt.Sprintf("gusts %.0f km/h", w.WindGustsKMH)
			}
			return ""
		},
	},
	{
		title: "Snow",
		detail: func(_ forecastSample, w mapdata.Weather) string {
			if w.SnowfallCM >= 0.5 {
				return fmt.Sprintf("%.1f cm/h", w.SnowfallCM)
			}
			return ""
		},
	},
	{
		title: "Rain likely",
		detail: func(_ forecastSample, w mapdata.Weather) string {
			if w.PrecipitationProbability >= 60 && w.PrecipitationMM >= 1 && w.SnowfallCM < 0.5 {
				return fmt.Sprintf("%d%%, %.1f mm/h", w.PrecipitationProbability, w.PrecipitationMM)
			}
			return ""
		},
	},
	{
		title: "Very cold",
		detail: func(_ forecastSample, w mapdata.Weather) string {
			if w.ApparentTemperature <= -10 {
				return fmt.Sprintf("feels like %.0f°C", w.ApparentTemperature)
			}
			return ""
		},
	},
	{
		title: "Hot",
		detail: func(_ forecastSample, w mapdata.Weather) string {
			if w.ApparentTemperature >= 30 {
				return fmt.Sprintf("feels like %.0f°C", w.ApparentTemperature)
			}
			return ""
		},
	},
	{
		title: "High UV",
		detail: func(_ forecastSample, w mapdata.Weather) string {
			if w.UVIndex >= 8 {
				return fmt.Sprintf("UV index %.0f", w.UVIndex)
			}
			return ""
		},
	},
	{
		title: "In darkness, bring a headlamp",
		detail: func(_ forecastSample, w mapdata.Weather) string {
			if !w.IsDay {
				return "dark"
			}
			return ""
		},
	},
}

// maxHazardOccurrences is the number of occurrences listed per hazard.
const maxHazardOccurrences = 3

// forecastWarnings lists the hazards in the forecast, with the local times
// and distances at which they occur.
func forecastWarnings(samples []forecastSample, weather []mapdata.Weather) []string {
	var warnings []string
	for _, hazard := range forecastHazards {
		var occurrences []string
		count := 0
		for i, sample := range samples {
			detail := hazard.detail(sample, weather[i])
			if detail == "" {
				continue
			}
			count++
			if len(occurrences) < maxHazardOccurrences {
				local := weather[i].Time.Add(weather[i].UTCOffset)
				occurrences = append(occurrences, fmt.Sprintf("%s at km %.1f (%s)", local.Format("Mon 15:04"), sample.distance/1000, detail))
			}
		}
		if count == 0 {
			continue
		}
		warning := hazard.title + ": " + strings.Join(occurrences, "; ")
		if count > len(occurrences) {
			warning += fmt.Sprintf(" and %d more", count-len(occurrences))
		}
		warnings = append(warnings, warning)
	}
	return warnings
}
