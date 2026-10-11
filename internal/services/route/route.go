package route

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/authn"
	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/routes"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"gitlab.com/greyxor/slogor"
	"google.golang.org/protobuf/proto"
)

func (s *Service) PreviewRoute(ctx context.Context, req *connect.Request[routesv1.PreviewRouteRequest]) (*connect.Response[routesv1.PreviewRouteResponse], error) {
	if _, ok := authn.GetUserID(ctx); !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	route := &routes.Route{
		Profile:   profileFromProto(req.Msg.GetProfile()),
		Source:    routes.SourceWaypoints,
		Waypoints: waypointsFromProto(req.Msg.GetWaypoints()),
	}
	legs, err := buildLegs(ctx, s.mapData, route.Waypoints, route.Profile, nil)
	if err != nil {
		return nil, s.toConnectError(err, "failed to compute route")
	}
	route.Legs = legs
	computeStats(route)
	return connect.NewResponse(routesv1.PreviewRouteResponse_builder{
		Route: routeToProto(route, req.Msg.GetIncludePoints()),
	}.Build()), nil
}

func (s *Service) CreateRoute(ctx context.Context, req *connect.Request[routesv1.CreateRouteRequest]) (*connect.Response[routesv1.CreateRouteResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	route := &routes.Route{
		UserID:      userID,
		Name:        strings.TrimSpace(req.Msg.GetName()),
		Description: req.Msg.GetDescription(),
		Profile:     routes.ProfileTrail,
	}
	switch req.Msg.WhichSource() {
	case routesv1.CreateRouteRequest_Waypoints_case:
		route.Source = routes.SourceWaypoints
		route.Profile = profileFromProto(req.Msg.GetWaypoints().GetProfile())
		route.Waypoints = waypointsFromProto(req.Msg.GetWaypoints().GetWaypoints())
		legs, err := buildLegs(ctx, s.mapData, route.Waypoints, route.Profile, nil)
		if err != nil {
			return nil, s.toConnectError(err, "failed to compute route")
		}
		route.Legs = legs
	case routesv1.CreateRouteRequest_Gpx_case:
		track, err := geo.ParseGPX(strings.NewReader(req.Msg.GetGpx()))
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if err := s.fillMissingElevations(ctx, track.Points); err != nil {
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
		route.Source = routes.SourceGPX
		route.Waypoints, route.Legs = legsFromTrack(track.Points)
		if route.Name == "" {
			route.Name = track.Name
		}
	case routesv1.CreateRouteRequest_ActivityId_case:
		activityID, err := store.StringToUUID(req.Msg.GetActivityId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid activity ID: %w", err))
		}
		activity, points, err := s.activityTrack(ctx, userID, activityID)
		if err != nil {
			return nil, err
		}
		route.Source = routes.SourceActivity
		route.Waypoints, route.Legs = legsFromTrack(points)
		if route.Name == "" {
			route.Name = activity.Name
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("one of waypoints, gpx or activity_id must be set"))
	}
	if route.Name == "" {
		route.Name = "Untitled route"
	}
	computeStats(route)
	created, err := s.routes.CreateRoute(ctx, route)
	if err != nil {
		return nil, s.toConnectError(err, "failed to create route")
	}
	return connect.NewResponse(routesv1.CreateRouteResponse_builder{
		Route: routeToProto(created, req.Msg.GetIncludePoints()),
	}.Build()), nil
}

func (s *Service) GetRoute(ctx context.Context, req *connect.Request[routesv1.GetRouteRequest]) (*connect.Response[routesv1.GetRouteResponse], error) {
	route, err := s.getRoute(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(routesv1.GetRouteResponse_builder{
		Route: routeToProto(route, req.Msg.GetIncludePoints()),
	}.Build()), nil
}

func (s *Service) ListRoutes(ctx context.Context, req *connect.Request[routesv1.ListRoutesRequest]) (*connect.Response[routesv1.ListRoutesResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	list, err := s.routes.ListRoutes(ctx, userID)
	if err != nil {
		return nil, s.toConnectError(err, "failed to list routes")
	}
	protoRoutes := make([]*routesv1.Route, len(list))
	for i, route := range list {
		protoRoutes[i] = routeToProto(route, false)
	}
	return connect.NewResponse(routesv1.ListRoutesResponse_builder{
		Routes: protoRoutes,
	}.Build()), nil
}

func (s *Service) UpdateRoute(ctx context.Context, req *connect.Request[routesv1.UpdateRouteRequest]) (*connect.Response[routesv1.UpdateRouteResponse], error) {
	route, err := s.getRoute(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if req.Msg.HasName() {
		route.Name = strings.TrimSpace(req.Msg.GetName())
		if route.Name == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name must not be empty"))
		}
	}
	if req.Msg.HasDescription() {
		route.Description = req.Msg.GetDescription()
	}
	if req.Msg.HasWaypoints() {
		waypoints := waypointsFromProto(req.Msg.GetWaypoints().GetWaypoints())
		profile := profileFromProto(req.Msg.GetWaypoints().GetProfile())
		legs, err := buildLegs(ctx, s.mapData, waypoints, profile, route.Legs)
		if err != nil {
			return nil, s.toConnectError(err, "failed to compute route")
		}
		route.Waypoints, route.Profile, route.Legs = waypoints, profile, legs
		computeStats(route)
	}
	updated, err := s.routes.UpdateRoute(ctx, route)
	if err != nil {
		return nil, s.toConnectError(err, "failed to update route")
	}
	return connect.NewResponse(routesv1.UpdateRouteResponse_builder{
		Route: routeToProto(updated, req.Msg.GetIncludePoints()),
	}.Build()), nil
}

func (s *Service) DeleteRoute(ctx context.Context, req *connect.Request[routesv1.DeleteRouteRequest]) (*connect.Response[routesv1.DeleteRouteResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	routeID, err := store.StringToUUID(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid route ID: %w", err))
	}
	if err := s.routes.DeleteRoute(ctx, userID, routeID); err != nil {
		return nil, s.toConnectError(err, "failed to delete route")
	}
	return connect.NewResponse(routesv1.DeleteRouteResponse_builder{}.Build()), nil
}

// getRoute gets one of the authenticated user's routes by ID.
func (s *Service) getRoute(ctx context.Context, id string) (*routes.Route, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	routeID, err := store.StringToUUID(id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid route ID: %w", err))
	}
	route, err := s.routes.GetRoute(ctx, userID, routeID)
	if err != nil {
		return nil, s.toConnectError(err, "failed to get route")
	}
	return route, nil
}

// activityTrack returns an activity and its GPS track, from its streams if
// synced or its summary polyline otherwise.
func (s *Service) activityTrack(ctx context.Context, userID, activityID pgtype.UUID) (*activities.Activity, []geo.Point, error) {
	activity, err := s.activities.GetActivity(ctx, userID, activityID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil, connect.NewError(connect.CodeNotFound, errors.New("activity not found"))
		}
		return nil, nil, s.toConnectError(err, "failed to get activity")
	}
	var points []geo.Point
	data, err := s.activities.GetActivityStreams(ctx, userID, activityID)
	switch {
	case err == nil:
		points, err = decodeStreams(data)
		if err != nil {
			return nil, nil, s.toConnectError(err, "failed to decode activity streams")
		}
	case errors.Is(err, store.ErrNotFound):
		if activity.SummaryPolyline != "" {
			points, err = geo.DecodePolyline(activity.SummaryPolyline)
			if err != nil {
				return nil, nil, s.toConnectError(err, "failed to decode activity polyline")
			}
		}
	default:
		return nil, nil, s.toConnectError(err, "failed to get activity streams")
	}
	if len(points) < 2 {
		return nil, nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("the activity has no GPS track"))
	}
	if err := s.fillMissingElevations(ctx, points); err != nil {
		return nil, nil, connect.NewError(connect.CodeUnavailable, err)
	}
	return activity, points, nil
}

// fillMissingElevations looks up elevations if any point lacks one.
func (s *Service) fillMissingElevations(ctx context.Context, points []geo.Point) error {
	for _, p := range points {
		if math.IsNaN(p.Ele) {
			return s.mapData.FillElevations(ctx, points, maxElevationLookups)
		}
	}
	return nil
}

// decodeStreams decodes stored activity streams into points.
func decodeStreams(data []byte) ([]geo.Point, error) {
	var streams activitiesv1.ActivityStreams
	if err := proto.Unmarshal(data, &streams); err != nil {
		return nil, fmt.Errorf("failed to decode streams: %w", err)
	}
	lats, lngs, alts := streams.GetLat(), streams.GetLng(), streams.GetAltitudeMeters()
	points := make([]geo.Point, 0, len(lats))
	for i := range lats {
		if i >= len(lngs) {
			break
		}
		ele := math.NaN()
		if i < len(alts) {
			ele = alts[i]
		}
		points = append(points, geo.Point{Lat: lats[i], Lng: lngs[i], Ele: ele})
	}
	return points, nil
}

func (s *Service) toConnectError(err error, msg string) error {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("route not found"))
	case errors.Is(err, errInvalidRoute):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	default:
		s.log.Error(msg, slogor.Err(err))
		return connect.NewError(connect.CodeInternal, fmt.Errorf("%s: %w", msg, err))
	}
}
