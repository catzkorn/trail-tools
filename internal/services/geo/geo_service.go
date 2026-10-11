// Package geo implements the GeoService, which provides map data for
// planning routes.
package geo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/authn"
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	"github.com/catzkorn/trail-tools/internal/gen/geo/v1/geov1connect"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/mapdata"
	"github.com/catzkorn/trail-tools/internal/services/route"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
)

var _ geov1connect.GeoServiceHandler = (*Service)(nil)

type MapData interface {
	Search(ctx context.Context, query string, limit int) ([]mapdata.Place, error)
	FindTrails(ctx context.Context, center geo.Point, radius float64) ([]mapdata.Element, error)
	FindFeatures(ctx context.Context, center geo.Point, radius float64, filters []string) ([]mapdata.Element, error)
	Elevations(ctx context.Context, points []geo.Point) ([]float64, error)
}

type TrackRepository interface {
	ListTracksInBounds(ctx context.Context, userID pgtype.UUID, bounds geo.Bounds, startDate, endDate time.Time) ([]activities.Track, error)
}

// Service implements API handlers for the geo service.
type Service struct {
	log     *slog.Logger
	mapData MapData
	tracks  TrackRepository
}

// NewService creates a new Service.
func NewService(log *slog.Logger, mapData MapData, tracks TrackRepository) *Service {
	return &Service{
		log:     log,
		mapData: mapData,
		tracks:  tracks,
	}
}

const (
	defaultRadiusMeters = 3000
	maxRadiusMeters     = 15000
)

func (s *Service) SearchPlaces(ctx context.Context, req *connect.Request[geov1.SearchPlacesRequest]) (*connect.Response[geov1.SearchPlacesResponse], error) {
	if _, ok := authn.GetUserID(ctx); !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	query := strings.TrimSpace(req.Msg.GetQuery())
	if query == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("query is required"))
	}
	limit := clamp(int(req.Msg.GetLimit()), 5, 20)
	places, err := s.mapData.Search(ctx, query, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	protoPlaces := make([]*geov1.Place, len(places))
	for i, p := range places {
		b := geov1.Place_builder{
			Name:     p.Name,
			Location: latLng(p.Lat, p.Lng),
			Category: p.Category,
			Type:     p.Type,
		}
		if len(p.BoundingBox) == 4 {
			b.Bounds = geov1.BoundingBox_builder{
				South: p.BoundingBox[0],
				North: p.BoundingBox[1],
				West:  p.BoundingBox[2],
				East:  p.BoundingBox[3],
			}.Build()
		}
		protoPlaces[i] = b.Build()
	}
	return connect.NewResponse(geov1.SearchPlacesResponse_builder{Places: protoPlaces}.Build()), nil
}

func (s *Service) FindTrails(ctx context.Context, req *connect.Request[geov1.FindTrailsRequest]) (*connect.Response[geov1.FindTrailsResponse], error) {
	if _, ok := authn.GetUserID(ctx); !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	center, radius, err := searchArea(req.Msg.GetCenter(), req.Msg.GetRadiusMeters())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	limit := clamp(int(req.Msg.GetLimit()), 50, 500)
	ways, err := s.mapData.FindTrails(ctx, center, radius)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	type trail struct {
		proto  *geov1.Trail
		length float64
	}
	var trails []trail
	for _, way := range ways {
		points := way.Points()
		if len(points) < 2 {
			continue
		}
		// Only count the part of the way inside the search area.
		var length float64
		for i := 1; i < len(points); i++ {
			if geo.Distance(center, geo.Interpolate(points[i-1], points[i], 0.5)) <= radius {
				length += geo.Distance(points[i-1], points[i])
			}
		}
		if length == 0 {
			continue
		}
		nearest, nearestDist := geo.NearestIndex(points, center)
		b := geov1.Trail_builder{
			OsmId:                    strconv.FormatInt(way.ID, 10),
			Name:                     way.Tags["name"],
			Highway:                  way.Tags["highway"],
			Surface:                  way.Tags["surface"],
			SacScale:                 way.Tags["sac_scale"],
			Terrain:                  route.TerrainToProto(geo.ClassifyTerrain(way.Tags)),
			LengthMeters:             math.Round(length),
			Start:                    latLng(points[0].Lat, points[0].Lng),
			End:                      latLng(points[len(points)-1].Lat, points[len(points)-1].Lng),
			NearestPoint:             latLng(points[nearest].Lat, points[nearest].Lng),
			DistanceFromCenterMeters: math.Round(nearestDist),
		}
		if req.Msg.GetIncludeGeometry() {
			for _, p := range points {
				b.Geometry = append(b.Geometry, latLng(p.Lat, p.Lng))
			}
		}
		trails = append(trails, trail{proto: b.Build(), length: length})
	}
	sort.SliceStable(trails, func(i, j int) bool { return trails[i].length > trails[j].length })
	result := make([]*geov1.Trail, 0, min(limit, len(trails)))
	for _, t := range trails[:min(limit, len(trails))] {
		result = append(result, t.proto)
	}
	return connect.NewResponse(geov1.FindTrailsResponse_builder{Trails: result}.Build()), nil
}

// poiFilters are the Overpass tag filters for each kind of point of
// interest.
var poiFilters = map[geov1.PointOfInterestKind][]string{
	geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_PEAK:      {`["natural"="peak"]`, `["natural"="volcano"]`},
	geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_WATER:     {`["amenity"="drinking_water"]`, `["natural"="spring"]`, `["man_made"="water_tap"]`},
	geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_PARKING:   {`["amenity"="parking"]`},
	geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_VIEWPOINT: {`["tourism"="viewpoint"]`},
	geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_SHELTER:   {`["amenity"="shelter"]`, `["tourism"="alpine_hut"]`, `["tourism"="wilderness_hut"]`},
	geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_TOILETS:   {`["amenity"="toilets"]`},
	geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_SADDLE:    {`["natural"="saddle"]`, `["mountain_pass"="yes"]`},
	geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_TRAILHEAD: {`["highway"="trailhead"]`},
}

// classifyPOI returns the kind of a point of interest from its tags.
func classifyPOI(tags map[string]string) geov1.PointOfInterestKind {
	switch {
	case tags["natural"] == "peak" || tags["natural"] == "volcano":
		return geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_PEAK
	case tags["amenity"] == "drinking_water" || tags["natural"] == "spring" || tags["man_made"] == "water_tap":
		return geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_WATER
	case tags["amenity"] == "parking":
		return geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_PARKING
	case tags["tourism"] == "viewpoint":
		return geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_VIEWPOINT
	case tags["amenity"] == "shelter" || tags["tourism"] == "alpine_hut" || tags["tourism"] == "wilderness_hut":
		return geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_SHELTER
	case tags["amenity"] == "toilets":
		return geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_TOILETS
	case tags["natural"] == "saddle" || tags["mountain_pass"] == "yes":
		return geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_SADDLE
	case tags["highway"] == "trailhead":
		return geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_TRAILHEAD
	}
	return geov1.PointOfInterestKind_POINT_OF_INTEREST_KIND_UNSPECIFIED
}

func (s *Service) FindPointsOfInterest(ctx context.Context, req *connect.Request[geov1.FindPointsOfInterestRequest]) (*connect.Response[geov1.FindPointsOfInterestResponse], error) {
	if _, ok := authn.GetUserID(ctx); !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	center, radius, err := searchArea(req.Msg.GetCenter(), req.Msg.GetRadiusMeters())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	limit := clamp(int(req.Msg.GetLimit()), 100, 500)
	kinds := req.Msg.GetKinds()
	if len(kinds) == 0 {
		for kind := range poiFilters {
			kinds = append(kinds, kind)
		}
	}
	wanted := map[geov1.PointOfInterestKind]bool{}
	var filters []string
	for _, kind := range kinds {
		f, ok := poiFilters[kind]
		if !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown kind %v", kind))
		}
		if !wanted[kind] {
			filters = append(filters, f...)
			wanted[kind] = true
		}
	}
	elements, err := s.mapData.FindFeatures(ctx, center, radius, filters)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	var points []*geov1.PointOfInterest
	for _, e := range elements {
		kind := classifyPOI(e.Tags)
		loc, ok := e.Location()
		if !ok || !wanted[kind] {
			continue
		}
		ele, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(e.Tags["ele"]), " m"), 64)
		points = append(points, geov1.PointOfInterest_builder{
			OsmId:                    strconv.FormatInt(e.ID, 10),
			Kind:                     kind,
			Name:                     e.Tags["name"],
			Location:                 latLng(loc.Lat, loc.Lng),
			ElevationMeters:          ele,
			DistanceFromCenterMeters: math.Round(geo.Distance(center, loc)),
		}.Build())
	}
	sort.SliceStable(points, func(i, j int) bool {
		return points[i].GetDistanceFromCenterMeters() < points[j].GetDistanceFromCenterMeters()
	})
	return connect.NewResponse(geov1.FindPointsOfInterestResponse_builder{
		Points: points[:min(limit, len(points))],
	}.Build()), nil
}

func (s *Service) GetElevation(ctx context.Context, req *connect.Request[geov1.GetElevationRequest]) (*connect.Response[geov1.GetElevationResponse], error) {
	if _, ok := authn.GetUserID(ctx); !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	if n := len(req.Msg.GetPoints()); n == 0 || n > 100 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("between 1 and 100 points are required"))
	}
	points := make([]geo.Point, len(req.Msg.GetPoints()))
	for i, p := range req.Msg.GetPoints() {
		points[i] = geo.Point{Lat: p.GetLat(), Lng: p.GetLng()}
	}
	elevations, err := s.mapData.Elevations(ctx, points)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return connect.NewResponse(geov1.GetElevationResponse_builder{ElevationsMeters: elevations}.Build()), nil
}

const (
	defaultCellSizeMeters = 100
	minCellSizeMeters     = 25
	maxHeatmapCells       = 4_000_000
)

func (s *Service) GetPersonalHeatmap(ctx context.Context, req *connect.Request[geov1.GetPersonalHeatmapRequest]) (*connect.Response[geov1.GetPersonalHeatmapResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	pb := req.Msg.GetBounds()
	bounds := geo.Bounds{MinLat: pb.GetSouth(), MinLng: pb.GetWest(), MaxLat: pb.GetNorth(), MaxLng: pb.GetEast()}
	if !req.Msg.HasBounds() || bounds.MinLat >= bounds.MaxLat || bounds.MinLng >= bounds.MaxLng {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("bounds with south < north and west < east are required"))
	}
	cellSize := req.Msg.GetCellSizeMeters()
	if cellSize == 0 {
		cellSize = defaultCellSizeMeters
	}
	if cellSize < minCellSizeMeters {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cell_size_meters must be at least %d", minCellSizeMeters))
	}
	limit := clamp(int(req.Msg.GetLimit()), 2000, 20000)
	var startDate, endDate time.Time
	var err error
	if req.Msg.GetStartDate() != "" {
		if startDate, err = store.ParseDate(req.Msg.GetStartDate()); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	if req.Msg.GetEndDate() != "" {
		if endDate, err = store.ParseDate(req.Msg.GetEndDate()); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	grid := geo.NewGrid(bounds, cellSize)
	if grid.Rows*grid.Cols > maxHeatmapCells {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("the area is too large for the cell size; use a smaller area or larger cells"))
	}
	tracks, err := s.tracks.ListTracksInBounds(ctx, userID, bounds, startDate, endDate)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list tracks: %w", err))
	}
	counts := map[int]int32{}
	for _, track := range tracks {
		points, err := route.TrackPoints(track)
		if err != nil {
			s.log.WarnContext(ctx, "Skipping undecodable track", slog.String("activity_id", store.UUIDToString(track.ActivityID)))
			continue
		}
		for cell := range grid.Cells(points) {
			counts[cell]++
		}
	}
	cells := make([]*geov1.HeatmapCell, 0, len(counts))
	for cell, count := range counts {
		lat, lng := grid.Center(cell)
		cells = append(cells, geov1.HeatmapCell_builder{
			Center:        latLng(lat, lng),
			ActivityCount: count,
		}.Build())
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].GetActivityCount() != cells[j].GetActivityCount() {
			return cells[i].GetActivityCount() > cells[j].GetActivityCount()
		}
		if cells[i].GetCenter().GetLat() != cells[j].GetCenter().GetLat() {
			return cells[i].GetCenter().GetLat() < cells[j].GetCenter().GetLat()
		}
		return cells[i].GetCenter().GetLng() < cells[j].GetCenter().GetLng()
	})
	return connect.NewResponse(geov1.GetPersonalHeatmapResponse_builder{
		Cells:          cells[:min(limit, len(cells))],
		CellSizeMeters: cellSize,
		ActivityCount:  int32(len(tracks)),
	}.Build()), nil
}

func searchArea(center *geov1.LatLng, radius float64) (geo.Point, float64, error) {
	if center == nil {
		return geo.Point{}, 0, errors.New("center is required")
	}
	if radius == 0 {
		radius = defaultRadiusMeters
	}
	if radius < 0 || radius > maxRadiusMeters {
		return geo.Point{}, 0, fmt.Errorf("radius_meters must be between 0 and %d", maxRadiusMeters)
	}
	return geo.Point{Lat: center.GetLat(), Lng: center.GetLng()}, radius, nil
}

// clamp returns def if v is 0, otherwise v limited to at most maxV.
func clamp(v, def, maxV int) int {
	if v <= 0 {
		return def
	}
	return min(v, maxV)
}

func latLng(lat, lng float64) *geov1.LatLng {
	return geov1.LatLng_builder{Lat: lat, Lng: lng}.Build()
}
