package services_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"github.com/catzkorn/trail-tools/internal/activities"
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
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
	preview, err := e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints:     latLngs(0, 0, 0.01, 0, 0.02, 0),
		IncludePoints: true,
	}.Build())
	if err != nil {
		t.Fatalf("PreviewRoute: %v", err)
	}
	stats := preview.GetRoute().GetStats()
	if !approx(stats.GetDistanceMeters(), 2224, 5) || !approx(stats.GetElevationGainMeters(), 20, 0.5) {
		t.Errorf("unexpected preview stats %v", stats)
	}
	if len(preview.GetRoute().GetPoints()) != 21 {
		t.Errorf("got %d points, want 21", len(preview.GetRoute().GetPoints()))
	}
	terrain := stats.GetTerrain()
	if len(terrain) != 1 || terrain[0].GetTerrain() != geov1.Terrain_TERRAIN_TRAIL || !approx(terrain[0].GetFraction(), 1, 0.01) {
		t.Errorf("unexpected terrain %v", terrain)
	}
	if preview.GetRoute().GetId() != "" {
		t.Error("preview should not be saved")
	}

	// Unroutable waypoints are an invalid argument.
	_, err = e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints: latLngs(85, 0, 85.01, 0),
	}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)
	if !strings.Contains(err.Error(), "waypoint 1 to 2") {
		t.Errorf("error should name the leg: %v", err)
	}

	// Create the route and move the last waypoint: only the last leg is
	// routed again.
	created, err := e.routes.CreateRoute(ctx, routesv1.CreateRouteRequest_builder{
		Name:      "Hill loop",
		Waypoints: routesv1.Waypoints_builder{Waypoints: latLngs(0, 0, 0.01, 0, 0.02, 0)}.Build(),
	}.Build())
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	routeID := created.GetRoute().GetId()
	if created.GetRoute().GetSource() != routesv1.RouteSource_ROUTE_SOURCE_WAYPOINTS ||
		created.GetRoute().GetProfile() != routesv1.RoutingProfile_ROUTING_PROFILE_TRAIL {
		t.Errorf("unexpected route %v", created.GetRoute())
	}
	callsBefore := e.router.routeCallCount()
	updated, err := e.routes.UpdateRoute(ctx, routesv1.UpdateRouteRequest_builder{
		Id:        routeID,
		Name:      new("Bigger hill loop"),
		Waypoints: routesv1.Waypoints_builder{Waypoints: latLngs(0, 0, 0.01, 0, 0.03, 0)}.Build(),
	}.Build())
	if err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}
	if calls := e.router.routeCallCount() - callsBefore; calls != 1 {
		t.Errorf("got %d routing calls, want 1", calls)
	}
	if updated.GetRoute().GetName() != "Bigger hill loop" || !approx(updated.GetRoute().GetStats().GetDistanceMeters(), 3336, 5) {
		t.Errorf("unexpected updated route %v", updated.GetRoute())
	}

	// Straight legs look up elevations.
	straight, err := e.routes.UpdateRoute(ctx, routesv1.UpdateRouteRequest_builder{
		Id: routeID,
		Waypoints: routesv1.Waypoints_builder{
			Waypoints: latLngs(0, 0, 0.01, 0, 0.03, 0),
			Profile:   routesv1.RoutingProfile_ROUTING_PROFILE_STRAIGHT,
		}.Build(),
		IncludePoints: true,
	}.Build())
	if err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}
	if got := straight.GetRoute().GetStats().GetElevationGainMeters(); !approx(got, 30, 0.5) {
		t.Errorf("got gain %v, want 30", got)
	}
	if len(straight.GetRoute().GetStats().GetTerrain()) != 1 ||
		straight.GetRoute().GetStats().GetTerrain()[0].GetTerrain() != geov1.Terrain_TERRAIN_UNSPECIFIED {
		t.Errorf("straight legs should have unknown terrain: %v", straight.GetRoute().GetStats().GetTerrain())
	}

	// Routes are private.
	list, err := e.routes.ListRoutes(ctx, routesv1.ListRoutesRequest_builder{}.Build())
	if err != nil {
		t.Fatalf("ListRoutes: %v", err)
	}
	if len(list.GetRoutes()) != 1 || len(list.GetRoutes()[0].GetPoints()) != 0 {
		t.Errorf("unexpected routes %v", list.GetRoutes())
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
	imported, err := e.routes.CreateRoute(ctx, routesv1.CreateRouteRequest_builder{
		Gpx: new(string(gpx)),
	}.Build())
	if err != nil {
		t.Fatalf("CreateRoute from GPX: %v", err)
	}
	if imported.GetRoute().GetName() != "Bigger hill loop" ||
		imported.GetRoute().GetSource() != routesv1.RouteSource_ROUTE_SOURCE_GPX ||
		!approx(imported.GetRoute().GetStats().GetDistanceMeters(), 3336, 5) {
		t.Errorf("unexpected imported route %v", imported.GetRoute())
	}
	// Imported tracks are split into editable legs about a kilometer long.
	if n := len(imported.GetRoute().GetWaypoints()); n < 3 || n > 5 {
		t.Errorf("got %d waypoints for a 3.3km import", n)
	}

	if _, err := e.routes.DeleteRoute(ctx, routesv1.DeleteRouteRequest_builder{Id: routeID}.Build()); err != nil {
		t.Fatalf("DeleteRoute: %v", err)
	}
	_, err = e.routes.GetRoute(ctx, routesv1.GetRouteRequest_builder{Id: routeID}.Build())
	wantCode(t, err, connect.CodeNotFound)
}

func profiles(p ...routesv1.RoutingProfile) []routesv1.RoutingProfile {
	return p
}

func wantLegProfiles(t *testing.T, route *routesv1.Route, want ...routesv1.RoutingProfile) {
	t.Helper()
	got := route.GetLegProfiles()
	if len(got) != len(want) {
		t.Fatalf("got leg profiles %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got leg profiles %v, want %v", got, want)
		}
	}
}

func TestRouteLegProfiles(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()
	const (
		trail    = routesv1.RoutingProfile_ROUTING_PROFILE_TRAIL
		shortest = routesv1.RoutingProfile_ROUTING_PROFILE_SHORTEST
		straight = routesv1.RoutingProfile_ROUTING_PROFILE_STRAIGHT
		heatmap  = routesv1.RoutingProfile_ROUTING_PROFILE_HEATMAP
		unset    = routesv1.RoutingProfile_ROUTING_PROFILE_UNSPECIFIED
	)

	// A trail leg followed by a straight leg: only the trail leg is routed.
	preview, err := e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints:   latLngs(0, 0, 0.01, 0, 0.02, 0),
		LegProfiles: profiles(trail, straight),
	}.Build())
	if err != nil {
		t.Fatalf("PreviewRoute: %v", err)
	}
	if calls := e.router.routeCallCount(); calls != 1 {
		t.Errorf("got %d routing calls, want 1", calls)
	}
	wantLegProfiles(t, preview.GetRoute(), trail, straight)
	stats := preview.GetRoute().GetStats()
	terrain := stats.GetTerrain()
	if len(terrain) != 2 || terrain[0].GetTerrain() != geov1.Terrain_TERRAIN_TRAIL || !approx(terrain[0].GetFraction(), 0.5, 0.01) ||
		terrain[1].GetTerrain() != geov1.Terrain_TERRAIN_UNSPECIFIED {
		t.Errorf("unexpected terrain %v", terrain)
	}

	// There must be one profile per leg.
	_, err = e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints:   latLngs(0, 0, 0.01, 0, 0.02, 0),
		LegProfiles: profiles(trail),
	}.Build())
	wantCode(t, err, connect.CodeInvalidArgument)
	if !strings.Contains(err.Error(), "one entry per leg") {
		t.Errorf("error should explain leg_profiles: %v", err)
	}

	// Unset leg profiles use the route's profile. Legs routed for the preview
	// are reused.
	created, err := e.routes.CreateRoute(ctx, routesv1.CreateRouteRequest_builder{
		Name: "Mixed",
		Waypoints: routesv1.Waypoints_builder{
			Waypoints:   latLngs(0, 0, 0.01, 0, 0.02, 0),
			Profile:     straight,
			LegProfiles: profiles(trail, unset),
		}.Build(),
	}.Build())
	if err != nil {
		t.Fatalf("CreateRoute: %v", err)
	}
	if calls := e.router.routeCallCount(); calls != 1 {
		t.Errorf("got %d routing calls, want 1", calls)
	}
	wantLegProfiles(t, created.GetRoute(), trail, straight)
	if created.GetRoute().GetProfile() != straight {
		t.Errorf("got profile %v, want STRAIGHT", created.GetRoute().GetProfile())
	}
	routeID := created.GetRoute().GetId()

	// Extending the route with a heatmap leg only routes the new leg. The user
	// has no activities, so it is routed like a trail leg.
	updated, err := e.routes.UpdateRoute(ctx, routesv1.UpdateRouteRequest_builder{
		Id: routeID,
		Waypoints: routesv1.Waypoints_builder{
			Waypoints:   latLngs(0, 0, 0.01, 0, 0.02, 0, 0.03, 0),
			LegProfiles: profiles(trail, straight, heatmap),
		}.Build(),
	}.Build())
	if err != nil {
		t.Fatalf("UpdateRoute: %v", err)
	}
	if calls := e.router.routeCallCount(); calls != 2 {
		t.Errorf("got %d routing calls, want 2", calls)
	}
	wantLegProfiles(t, updated.GetRoute(), trail, straight, heatmap)
	if !approx(updated.GetRoute().GetStats().GetDistanceMeters(), 3336, 5) {
		t.Errorf("unexpected distance %v", updated.GetRoute().GetStats().GetDistanceMeters())
	}

	// Previewing changes to the saved route reuses its legs and only routes
	// the changed leg, without saving.
	preview, err = e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints:   latLngs(0, 0, 0.01, 0, 0.02, 0, 0.03, 0),
		LegProfiles: profiles(trail, shortest, heatmap),
		RouteId:     routeID,
	}.Build())
	if err != nil {
		t.Fatalf("PreviewRoute: %v", err)
	}
	if calls := e.router.routeCallCount(); calls != 3 {
		t.Errorf("got %d routing calls, want 3", calls)
	}
	wantLegProfiles(t, preview.GetRoute(), trail, shortest, heatmap)
	got, err := e.routes.GetRoute(ctx, routesv1.GetRouteRequest_builder{Id: routeID}.Build())
	if err != nil {
		t.Fatalf("GetRoute: %v", err)
	}
	wantLegProfiles(t, got.GetRoute(), trail, straight, heatmap)

	// The way back of an out and back route reuses the outbound legs.
	before := e.router.routeCallCount()
	outAndBack, err := e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints: latLngs(1, 0, 1.01, 0, 1.02, 0, 1.01, 0, 1, 0),
	}.Build())
	if err != nil {
		t.Fatalf("PreviewRoute: %v", err)
	}
	if calls := e.router.routeCallCount() - before; calls != 2 {
		t.Errorf("got %d routing calls, want 2", calls)
	}
	stats = outAndBack.GetRoute().GetStats()
	if !approx(stats.GetDistanceMeters(), 4448, 5) || !approx(stats.GetElevationGainMeters(), 20, 0.5) || !approx(stats.GetElevationLossMeters(), 20, 0.5) {
		t.Errorf("unexpected out and back stats %v", stats)
	}
	if terrain := stats.GetTerrain(); len(terrain) != 1 || terrain[0].GetTerrain() != geov1.Terrain_TERRAIN_TRAIL {
		t.Errorf("unexpected out and back terrain %v", terrain)
	}

	// Previewing another user's route is not allowed.
	_, err = e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints: latLngs(0, 0, 0.01, 0),
		RouteId:   "00000000-0000-0000-0000-000000000000",
	}.Build())
	wantCode(t, err, connect.CodeNotFound)
}

func TestHeatmapRouting(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()

	// The user has run a path between two points.
	from, to := geo.Point{Lat: 0, Lng: 0}, geo.Point{Lat: 0.01, Lng: 0}
	if _, err := e.activities.UpsertActivity(ctx, e.userID, &activities.ActivitySummary{
		StravaID:        1,
		Name:            "Usual loop",
		SportType:       "TrailRun",
		StartTime:       time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC),
		StartDateLocal:  time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		Timezone:        "(GMT+00:00) Europe/London",
		SummaryPolyline: geo.EncodePolyline(fakeRoutePath(from, to, 0.003)),
		RawJSON:         []byte("{}"),
	}); err != nil {
		t.Fatal(err)
	}

	// The heatmap profile asks routing to prefer the user's activities.
	preview, err := e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints: latLngs(from.Lat, from.Lng, to.Lat, to.Lng),
		Profile:   routesv1.RoutingProfile_ROUTING_PROFILE_HEATMAP,
	}.Build())
	if err != nil {
		t.Fatalf("PreviewRoute: %v", err)
	}
	wantLegProfiles(t, preview.GetRoute(), routesv1.RoutingProfile_ROUTING_PROFILE_HEATMAP)
	prefer := e.router.lastPrefer()
	if len(prefer) != 1 || len(prefer[0]) < 3 {
		t.Fatalf("got preferred paths %v, want the activity's track", prefer)
	}
	var maxLng float64
	for _, p := range prefer[0] {
		maxLng = max(maxLng, p.Lng)
	}
	if !approx(maxLng, 0.003, 0.0001) {
		t.Errorf("preferred path reaches longitude %v, want 0.003", maxLng)
	}

	// The trail profile doesn't.
	if _, err := e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints: latLngs(from.Lat, from.Lng, 0.02, 0),
	}.Build()); err != nil {
		t.Fatalf("PreviewRoute: %v", err)
	}
	if prefer := e.router.lastPrefer(); len(prefer) != 0 {
		t.Errorf("trail profile preferred %d paths", len(prefer))
	}

	// Away from the user's activities, there is nothing to prefer.
	if _, err := e.routes.PreviewRoute(ctx, routesv1.PreviewRouteRequest_builder{
		Waypoints: latLngs(10, 10, 10.01, 10),
		Profile:   routesv1.RoutingProfile_ROUTING_PROFILE_HEATMAP,
	}.Build()); err != nil {
		t.Fatalf("PreviewRoute: %v", err)
	}
	if prefer := e.router.lastPrefer(); len(prefer) != 0 {
		t.Errorf("got %d preferred paths away from the activities", len(prefer))
	}
}
