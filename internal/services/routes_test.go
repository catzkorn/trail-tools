package services_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
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
