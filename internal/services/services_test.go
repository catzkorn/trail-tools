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

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/gen/activities/v1/activitiesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/routes/v1/routesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/strava/v1/stravav1connect"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/mapdata"
	"github.com/catzkorn/trail-tools/internal/routes"
	"github.com/catzkorn/trail-tools/internal/services/activity"
	"github.com/catzkorn/trail-tools/internal/services/auth"
	"github.com/catzkorn/trail-tools/internal/services/route"
	stravasvc "github.com/catzkorn/trail-tools/internal/services/strava"
	"github.com/catzkorn/trail-tools/internal/services/user"
	"github.com/catzkorn/trail-tools/internal/store/storetest"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/jackc/pgx/v5/pgtype"
)

// env is a running server for a test, with clients authenticated as a user.
type env struct {
	t          *testing.T
	userID     pgtype.UUID
	token      string
	srv        *httptest.Server
	mapData    *fakeMapData
	activities *activities.Repository
	syncer     *stravasvc.Syncer
	strava     *fakeStrava

	users     usersv1connect.UserServiceClient
	activity  activitiesv1connect.ActivityServiceClient
	stravaAPI stravav1connect.StravaServiceClient
	routes    routesv1connect.RouteServiceClient
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
	e := &env{t: t, mapData: &fakeMapData{}, activities: activityRepo, strava: newFakeStrava()}
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
	routeService := route.NewService(log, routeRepo, activityRepo, e.mapData)

	mux := http.NewServeMux()
	mux.Handle(usersv1connect.NewUserServiceHandler(userService))
	mux.Handle(authv1connect.NewAuthServiceHandler(authService))
	mux.Handle(activitiesv1connect.NewActivityServiceHandler(activityService))
	mux.Handle(stravav1connect.NewStravaServiceHandler(stravaService))
	mux.Handle(routesv1connect.NewRouteServiceHandler(routeService))
	mux.Handle("/routes/", routeService.GPXHandler(http.NotFoundHandler()))
	handler, err := authn.NewAuthnMiddleware(ctx, log, userRepo, mux)
	if err != nil {
		t.Fatal(err)
	}
	e.srv = httptest.NewServer(handler)
	t.Cleanup(e.srv.Close)

	client := e.srv.Client()
	opts := connect.WithInterceptors(bearerInterceptor(e.token))
	e.users = usersv1connect.NewUserServiceClient(client, e.srv.URL, opts)
	e.activity = activitiesv1connect.NewActivityServiceClient(client, e.srv.URL, opts)
	e.stravaAPI = stravav1connect.NewStravaServiceClient(client, e.srv.URL, opts)
	e.routes = routesv1connect.NewRouteServiceClient(client, e.srv.URL, opts)
	return e
}

// bootstrapToken is the admin bootstrap token in tests.
const bootstrapToken = "bootstrap-secret"

// bearerInterceptor authenticates requests with an API token.
func bearerInterceptor(token string) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	})
}

// wantCode fails the test unless err is a connect error with the code.
func wantCode(t *testing.T, err error, code connect.Code) {
	t.Helper()
	if connect.CodeOf(err) != code {
		t.Fatalf("got error %v, want code %v", err, code)
	}
}

// fakeMapData routes legs as straight lines with 11 points, where the
// elevation is 1000 times the latitude and the terrain is trail.
type fakeMapData struct {
	mu         sync.Mutex
	routeCalls int
}

func (f *fakeMapData) Route(_ context.Context, from, to geo.Point, profile string) (*mapdata.RoutedPath, error) {
	f.mu.Lock()
	f.routeCalls++
	f.mu.Unlock()
	if from.Lat > 80 {
		return nil, fmt.Errorf("from-position not mapped")
	}
	path := &mapdata.RoutedPath{}
	for i := range 11 {
		p := geo.Interpolate(from, to, float64(i)/10)
		p.Ele = p.Lat * 1000
		path.Points = append(path.Points, p)
		path.Terrains = append(path.Terrains, geo.TerrainTrail)
	}
	path.Terrains[10] = geo.TerrainUnknown
	return path, nil
}

func (f *fakeMapData) routeCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.routeCalls
}

func (f *fakeMapData) FillElevations(_ context.Context, points []geo.Point, _ int) error {
	for i := range points {
		points[i].Ele = points[i].Lat * 1000
	}
	return nil
}

func approx(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}
