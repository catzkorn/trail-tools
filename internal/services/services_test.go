// Package services_test contains integration tests of the API services,
// running against a real database through the HTTP handlers and
// authentication middleware.
package services_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/gen/activities/v1/activitiesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/routes/v1/routesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/routing/v1/routingv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/strava/v1/stravav1connect"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/routes"
	"github.com/catzkorn/trail-tools/internal/routing"
	"github.com/catzkorn/trail-tools/internal/services/activity"
	"github.com/catzkorn/trail-tools/internal/services/auth"
	mapdatasvc "github.com/catzkorn/trail-tools/internal/services/mapdata"
	"github.com/catzkorn/trail-tools/internal/services/route"
	stravasvc "github.com/catzkorn/trail-tools/internal/services/strava"
	"github.com/catzkorn/trail-tools/internal/services/user"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/store/storetest"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/jackc/pgx/v5/pgtype"
)

// env is a running server for a test, with clients authenticated as a user.
type env struct {
	t          *testing.T
	db         *store.DB
	userID     pgtype.UUID
	token      string
	srv        *httptest.Server
	mapData    *fakeMapData
	router     *fakeRouter
	activities *activities.Repository
	syncer     *stravasvc.Syncer
	strava     *fakeStrava

	users      usersv1connect.UserServiceClient
	activity   activitiesv1connect.ActivityServiceClient
	stravaAPI  stravav1connect.StravaServiceClient
	routes     routesv1connect.RouteServiceClient
	mapDataAPI routingv1connect.MapDataServiceClient
	imports    *fakeImportRunner
}

func newEnv(t *testing.T) *env {
	t.Helper()
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := storetest.New(t)
	userRepo, err := users.NewRepository(log, db)
	if err != nil {
		t.Fatal(err)
	}
	activityRepo, err := activities.NewRepository(log, db)
	if err != nil {
		t.Fatal(err)
	}
	routeRepo, err := routes.NewRepository(log, db)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, db: db, mapData: &fakeMapData{}, router: &fakeRouter{}, activities: activityRepo, strava: newFakeStrava()}
	e.userID = storetest.NewUser(t, db)
	_, e.token, err = userRepo.CreateAPIToken(ctx, e.userID, "test")
	if err != nil {
		t.Fatal(err)
	}

	syncCtx, cancel := context.WithCancel(ctx)
	e.syncer = stravasvc.NewSyncer(syncCtx, log, activityRepo, e.strava, true)
	t.Cleanup(func() {
		cancel()
		e.syncer.Wait()
	})

	userService := user.NewService(log, userRepo)
	authService, err := auth.NewService(log, userRepo, "http://localhost:8080", bootstrapToken)
	if err != nil {
		t.Fatal(err)
	}
	activityService := activity.NewService(log, activityRepo)
	stravaService := stravasvc.NewService(log, activityRepo, e.syncer, nil)
	routeService := route.NewService(log, routeRepo, activityRepo, e.router, e.mapData)

	mux := http.NewServeMux()
	server := connect.NewServer()
	usersv1connect.RegisterUserServiceHandler(server, userService)
	authv1connect.RegisterAuthServiceHandler(server, authService)
	activitiesv1connect.RegisterActivityServiceHandler(server, activityService)
	stravav1connect.RegisterStravaServiceHandler(server, stravaService)
	routesv1connect.RegisterRouteServiceHandler(server, routeService)
	connecthttp.Mount(mux, server)

	mux.Handle("/routes/", routeService.GPXHandler(http.NotFoundHandler()))
	e.imports = &fakeImportRunner{}
	srv2 := connect.NewServer()
	routingv1connect.RegisterMapDataServiceHandler(srv2, mapdatasvc.NewService(log, db, fakeExtracts{}, e.imports))
	connecthttp.Mount(mux, srv2)
	handler, err := authn.NewAuthnMiddleware(ctx, log, userRepo, mux)
	if err != nil {
		t.Fatal(err)
	}
	e.srv = httptest.NewServer(handler)
	t.Cleanup(e.srv.Close)

	client := e.srv.Client()
	authed := connect.NewClient(connecthttp.NewTransport(client, e.srv.URL), bearerInterceptor(e.token))
	e.users = usersv1connect.NewUserServiceClient(authed)
	e.activity = activitiesv1connect.NewActivityServiceClient(authed)
	e.stravaAPI = stravav1connect.NewStravaServiceClient(authed)
	e.routes = routesv1connect.NewRouteServiceClient(authed)
	e.mapDataAPI = routingv1connect.NewMapDataServiceClient(authed)
	return e
}

// bootstrapToken is the admin bootstrap token in tests.
const bootstrapToken = "bootstrap-secret"

// bearerInterceptor authenticates requests with an API token.
func bearerInterceptor(token string) connect.ClientInterceptor {
	return func(next connect.ClientFunc) connect.ClientFunc {
		return func(ctx context.Context, spec connect.Spec) (connect.ClientStream, error) {
			if info, ok := connect.CallInfoForClientContext(ctx); ok {
				info.RequestHeader().Set("Authorization", "Bearer "+token)
			}
			return next(ctx, spec)
		}
	}
}

// wantCode fails the test unless err is a connect error with the code.
func wantCode(t *testing.T, err error, code connect.Code) {
	t.Helper()
	if connect.CodeOf(err) != code {
		t.Fatalf("got error %v, want code %v", err, code)
	}
}

// fakeRouter routes legs as straight lines with 11 points, where the
// elevation is 1000 times the latitude and the terrain is trail. Points north
// of 80 degrees can't be routed. It records the preferred paths of the last
// request.
type fakeRouter struct {
	mu         sync.Mutex
	routeCalls int
	prefer     [][]geo.Point
}

// fakeRoutePath returns the points of a path that bows east by bow degrees
// in the middle.
func fakeRoutePath(from, to geo.Point, bow float64) []geo.Point {
	var points []geo.Point
	for i := range 11 {
		f := float64(i) / 10
		p := geo.Interpolate(from, to, f)
		p.Lng += bow * math.Sin(math.Pi*f)
		p.Ele = p.Lat * 1000
		points = append(points, p)
	}
	return points
}

func (f *fakeRouter) Route(_ context.Context, req routing.Request) (*routing.Path, error) {
	f.mu.Lock()
	f.routeCalls++
	f.prefer = req.Prefer
	f.mu.Unlock()
	if req.From.Lat > 80 {
		return nil, fmt.Errorf("%w: there is no mapped path near the point", routing.ErrNotRoutable)
	}
	path := &routing.Path{Points: fakeRoutePath(req.From, req.To, 0)}
	for range path.Points {
		path.Terrains = append(path.Terrains, geo.TerrainTrail)
	}
	path.Terrains[10] = geo.TerrainUnknown
	return path, nil
}

// Regions covers everything up to latitude 80, matching where Route finds
// paths.
func (f *fakeRouter) Regions(context.Context) ([]geo.Bounds, error) {
	return []geo.Bounds{{MinLat: -90, MinLng: -180, MaxLat: 80, MaxLng: 180}}, nil
}

func (f *fakeRouter) routeCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.routeCalls
}

func (f *fakeRouter) lastPrefer() [][]geo.Point {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.prefer
}

// fakeMapData looks up elevations as 1000 times the latitude.
type fakeMapData struct{}

func (f *fakeMapData) FillElevations(_ context.Context, points []geo.Point, _ int) error {
	for i := range points {
		points[i].Ele = points[i].Lat * 1000
	}
	return nil
}

func approx(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}
