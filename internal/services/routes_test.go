package services_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func latLngs(coords ...float64) []*geov1.LatLng {
	var points []*geov1.LatLng
	for i := 0; i+1 < len(coords); i += 2 {
		points = append(points, geov1.LatLng_builder{Lat: coords[i], Lng: coords[i+1]}.Build())
	}
	return points
}

func TestRoutes(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()

	// Preview a route over 3 waypoints, climbing 2m per 0.001 degrees.
	preview, err := e.routes.PreviewRoute(ctx, connect.NewRequest(routesv1.PreviewRouteRequest_builder{
		Waypoints:     latLngs(0, 0, 0.01, 0, 0.02, 0),
		IncludePoints: true,
	}.Build()))
	if err != nil {
		t.Fatalf("PreviewRoute: %v", err)
	}
	stats := preview.Msg.GetRoute().GetStats()
	if !approx(stats.GetDistanceMeters(), 2224, 5) || !approx(stats.GetElevationGainMeters(), 20, 0.5) {
		t.Errorf("unexpected preview stats %v", stats)
	}
	if len(preview.Msg.GetRoute().GetPoints()) != 21 {
		t.Errorf("got %d points, want 21", len(preview.Msg.GetRoute().GetPoints()))
	}
	terrain := stats.GetTerrain()
	if len(terrain) != 1 || terrain[0].GetTerrain() != geov1.Terrain_TERRAIN_TRAIL || !approx(terrain[0].GetFraction(), 1, 0.01) {
		t.Errorf("unexpected terrain %v", terrain)
	}
	if preview.Msg.GetRoute().GetId() != "" {
		t.Error("preview should not be saved")
	}

	// Unroutable waypoints are an invalid argument.
	_, err = e.routes.PreviewRoute(ctx, connect.NewRequest(routesv1.PreviewRouteRequest_builder{
		Waypoints: latLngs(85, 0, 85.01, 0),
	}.Build()))
	wantCode(t, err, connect.CodeInvalidArgument)
	if !strings.Contains(err.Error(), "waypoint 1 to 2") {
		t.Errorf("error should name the leg: %v", err)
	}

	// Create the route and move the last waypoint: only the last leg is
	// routed again.
	created, err := e.routes.CreateRoute(ctx, connect.NewRequest(routesv1.CreateRouteRequest_builder{
		Name:      "Hill loop",
		Waypoints: routesv1.Waypoints_builder{Waypoints: latLngs(0, 0, 0.01, 0, 0.02, 0)}.Build(),
	}.Build()))
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	routeID := created.Msg.GetRoute().GetId()
	if created.Msg.GetRoute().GetSource() != routesv1.RouteSource_ROUTE_SOURCE_WAYPOINTS ||
		created.Msg.GetRoute().GetProfile() != routesv1.RoutingProfile_ROUTING_PROFILE_TRAIL {
		t.Errorf("unexpected route %v", created.Msg.GetRoute())
	}
	callsBefore := e.mapData.routeCallCount()
	updated, err := e.routes.UpdateRoute(ctx, connect.NewRequest(routesv1.UpdateRouteRequest_builder{
		Id:        routeID,
		Name:      new("Bigger hill loop"),
		Waypoints: routesv1.Waypoints_builder{Waypoints: latLngs(0, 0, 0.01, 0, 0.03, 0)}.Build(),
	}.Build()))
	if err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}
	if calls := e.mapData.routeCallCount() - callsBefore; calls != 1 {
		t.Errorf("got %d routing calls, want 1", calls)
	}
	if updated.Msg.GetRoute().GetName() != "Bigger hill loop" || !approx(updated.Msg.GetRoute().GetStats().GetDistanceMeters(), 3336, 5) {
		t.Errorf("unexpected updated route %v", updated.Msg.GetRoute())
	}

	// Straight legs look up elevations.
	straight, err := e.routes.UpdateRoute(ctx, connect.NewRequest(routesv1.UpdateRouteRequest_builder{
		Id: routeID,
		Waypoints: routesv1.Waypoints_builder{
			Waypoints: latLngs(0, 0, 0.01, 0, 0.03, 0),
			Profile:   routesv1.RoutingProfile_ROUTING_PROFILE_STRAIGHT,
		}.Build(),
		IncludePoints: true,
	}.Build()))
	if err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}
	if got := straight.Msg.GetRoute().GetStats().GetElevationGainMeters(); !approx(got, 30, 0.5) {
		t.Errorf("got gain %v, want 30", got)
	}
	if len(straight.Msg.GetRoute().GetStats().GetTerrain()) != 1 ||
		straight.Msg.GetRoute().GetStats().GetTerrain()[0].GetTerrain() != geov1.Terrain_TERRAIN_UNSPECIFIED {
		t.Errorf("straight legs should have unknown terrain: %v", straight.Msg.GetRoute().GetStats().GetTerrain())
	}

	// Routes are private.
	list, err := e.routes.ListRoutes(ctx, connect.NewRequest(routesv1.ListRoutesRequest_builder{}.Build()))
	if err != nil {
		t.Fatalf("ListRoutes: %v", err)
	}
	if len(list.Msg.GetRoutes()) != 1 || len(list.Msg.GetRoutes()[0].GetPoints()) != 0 {
		t.Errorf("unexpected routes %v", list.Msg.GetRoutes())
	}

	// Download as GPX and import it again.
	req, err := http.NewRequest(http.MethodGet, e.srv.URL+"/routes/"+routeID+".gpx", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+e.token)
	resp, err := e.srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	gpx, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(gpx), "<trkpt") ||
		!strings.Contains(resp.Header.Get("Content-Disposition"), "Bigger-hill-loop.gpx") {
		t.Fatalf("unexpected GPX response %d %s", resp.StatusCode, gpx)
	}
	imported, err := e.routes.CreateRoute(ctx, connect.NewRequest(routesv1.CreateRouteRequest_builder{
		Gpx: new(string(gpx)),
	}.Build()))
	if err != nil {
		t.Fatalf("CreateRoute from GPX: %v", err)
	}
	if imported.Msg.GetRoute().GetName() != "Bigger hill loop" ||
		imported.Msg.GetRoute().GetSource() != routesv1.RouteSource_ROUTE_SOURCE_GPX ||
		!approx(imported.Msg.GetRoute().GetStats().GetDistanceMeters(), 3336, 5) {
		t.Errorf("unexpected imported route %v", imported.Msg.GetRoute())
	}
	// Imported tracks are split into editable legs about a kilometer long.
	if n := len(imported.Msg.GetRoute().GetWaypoints()); n < 3 || n > 5 {
		t.Errorf("got %d waypoints for a 3.3km import", n)
	}

	if _, err := e.routes.DeleteRoute(ctx, connect.NewRequest(routesv1.DeleteRouteRequest_builder{Id: routeID}.Build())); err != nil {
		t.Fatalf("DeleteRoute: %v", err)
	}
	_, err = e.routes.GetRoute(ctx, connect.NewRequest(routesv1.GetRouteRequest_builder{Id: routeID}.Build()))
	wantCode(t, err, connect.CodeNotFound)
}

func TestRouteForecast(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()

	// A 22km route north along a meridian, climbing to 200m.
	created, err := e.routes.CreateRoute(ctx, connect.NewRequest(routesv1.CreateRouteRequest_builder{
		Name:      "Long day",
		Waypoints: routesv1.Waypoints_builder{Waypoints: latLngs(0, 0, 0.2, 0)}.Build(),
	}.Build()))
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	start := time.Now().Add(24 * time.Hour).UTC().Truncate(24 * time.Hour).Add(9 * time.Hour)
	resp, err := e.routes.GetRouteForecast(ctx, connect.NewRequest(routesv1.GetRouteForecastRequest_builder{
		RouteId:            new(created.Msg.GetRoute().GetId()),
		StartTime:          timestamppb.New(start),
		SpeedKmh:           10,
		ClimbMetersPerHour: 400,
	}.Build()))
	if err != nil {
		t.Fatalf("GetRouteForecast: %v", err)
	}
	// 22.2km at 10km/h plus 200m at 400m/h is 2.72h.
	if got := time.Duration(resp.Msg.GetDurationSeconds()) * time.Second; !approx(got.Hours(), 2.72, 0.02) {
		t.Errorf("got duration %v", got)
	}
	var kinds []routesv1.ForecastPointKind
	for _, p := range resp.Msg.GetPoints() {
		kinds = append(kinds, p.GetKind())
	}
	// The high point is the finish, so it isn't repeated.
	want := []routesv1.ForecastPointKind{
		routesv1.ForecastPointKind_FORECAST_POINT_KIND_START,
		routesv1.ForecastPointKind_FORECAST_POINT_KIND_INTERVAL,
		routesv1.ForecastPointKind_FORECAST_POINT_KIND_INTERVAL,
		routesv1.ForecastPointKind_FORECAST_POINT_KIND_FINISH,
	}
	if len(kinds) != len(want) {
		t.Fatalf("got kinds %v, want %v", kinds, want)
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Fatalf("got kinds %v, want %v", kinds, want)
		}
	}
	second := resp.Msg.GetPoints()[1]
	if !second.GetTime().AsTime().Equal(start.Add(time.Hour)) || second.GetWeatherDescription() != "Thunderstorm" {
		t.Errorf("unexpected second point %v", second)
	}
	if len(resp.Msg.GetWarnings()) != 1 || !strings.HasPrefix(resp.Msg.GetWarnings()[0], "Thunderstorms possible: ") {
		t.Errorf("unexpected warnings %v", resp.Msg.GetWarnings())
	}

	// A GPX file or a list of coordinates can be forecast too.
	_, err = e.routes.GetRouteForecast(ctx, connect.NewRequest(routesv1.GetRouteForecastRequest_builder{
		Path:      routesv1.Path_builder{Points: latLngs(0, 0, 0.05, 0)}.Build(),
		StartTime: timestamppb.New(start),
		SpeedKmh:  8,
	}.Build()))
	if err != nil {
		t.Fatalf("GetRouteForecast with path: %v", err)
	}

	// Forecasts are only available for the next 2 weeks.
	_, err = e.routes.GetRouteForecast(ctx, connect.NewRequest(routesv1.GetRouteForecastRequest_builder{
		RouteId:   new(created.Msg.GetRoute().GetId()),
		StartTime: timestamppb.New(time.Now().AddDate(0, 1, 0)),
		SpeedKmh:  10,
	}.Build()))
	wantCode(t, err, connect.CodeInvalidArgument)
}
