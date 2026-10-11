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
	"time"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/gen/activities/v1/activitiesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/geo/v1/geov1connect"
	"github.com/catzkorn/trail-tools/internal/gen/races/v1/racesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/routes/v1/routesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/routing/v1/routingv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/strava/v1/stravav1connect"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/livetrack"
	"github.com/catzkorn/trail-tools/internal/mapdata"
	"github.com/catzkorn/trail-tools/internal/races"
	"github.com/catzkorn/trail-tools/internal/routes"
	"github.com/catzkorn/trail-tools/internal/routing"
	"github.com/catzkorn/trail-tools/internal/services/activity"
	"github.com/catzkorn/trail-tools/internal/services/auth"
	geosvc "github.com/catzkorn/trail-tools/internal/services/geo"
	mapdatasvc "github.com/catzkorn/trail-tools/internal/services/mapdata"
	"github.com/catzkorn/trail-tools/internal/services/race"
	"github.com/catzkorn/trail-tools/internal/services/raceplan"
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
	liveTrack  *fakeLiveTrack

	users      usersv1connect.UserServiceClient
	activity   activitiesv1connect.ActivityServiceClient
	stravaAPI  stravav1connect.StravaServiceClient
	routes     routesv1connect.RouteServiceClient
	geo        geov1connect.GeoServiceClient
	mapDataAPI routingv1connect.MapDataServiceClient
	imports    *fakeImportRunner
	races      racesv1connect.RaceServiceClient
	racePlans  racesv1connect.RacePlanServiceClient
	anonPlans  racesv1connect.RacePlanServiceClient
	otherRaces racesv1connect.RaceServiceClient
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
	raceRepo, err := races.NewRepository(log, db)
	if err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, db: db, mapData: &fakeMapData{}, router: &fakeRouter{}, activities: activityRepo, strava: newFakeStrava()}
	e.userID = storetest.NewUser(t, db)
	_, e.token, err = userRepo.CreateAPIToken(ctx, e.userID, "test")
	if err != nil {
		t.Fatal(err)
	}
	otherUser := storetest.NewUser(t, db)
	_, otherToken, err := userRepo.CreateAPIToken(ctx, otherUser, "other")
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
	geoService := geosvc.NewService(log, e.mapData, activityRepo)
	raceService := race.NewService(log, raceRepo, routeRepo)
	e.liveTrack = &fakeLiveTrack{}
	racePlanService := raceplan.NewService(log, raceRepo, routeRepo, e.liveTrack, "https://trail.example")

	mux := http.NewServeMux()
	mux.Handle(usersv1connect.NewUserServiceHandler(userService))
	mux.Handle(authv1connect.NewAuthServiceHandler(authService))
	mux.Handle(activitiesv1connect.NewActivityServiceHandler(activityService))
	mux.Handle(stravav1connect.NewStravaServiceHandler(stravaService))
	mux.Handle(routesv1connect.NewRouteServiceHandler(routeService))
	mux.Handle(geov1connect.NewGeoServiceHandler(geoService))
	mux.Handle(racesv1connect.NewRaceServiceHandler(raceService))
	mux.Handle(racesv1connect.NewRacePlanServiceHandler(racePlanService))
	mux.Handle("/routes/", routeService.GPXHandler(http.NotFoundHandler()))
	e.imports = &fakeImportRunner{}
	mux.Handle(routingv1connect.NewMapDataServiceHandler(mapdatasvc.NewService(log, db, fakeExtracts{}, e.imports)))
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
	e.geo = geov1connect.NewGeoServiceClient(client, e.srv.URL, opts)
	e.mapDataAPI = routingv1connect.NewMapDataServiceClient(client, e.srv.URL, opts)
	e.races = racesv1connect.NewRaceServiceClient(client, e.srv.URL, opts)
	e.racePlans = racesv1connect.NewRacePlanServiceClient(client, e.srv.URL, opts)
	e.anonPlans = racesv1connect.NewRacePlanServiceClient(client, e.srv.URL)
	e.otherRaces = racesv1connect.NewRaceServiceClient(client, e.srv.URL, connect.WithInterceptors(bearerInterceptor(otherToken)))
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
type fakeMapData struct {
	mu        sync.Mutex
	forecasts []mapdata.ForecastQuery
}

func (f *fakeMapData) FillElevations(_ context.Context, points []geo.Point, _ int) error {
	for i := range points {
		points[i].Ele = points[i].Lat * 1000
	}
	return nil
}

func (f *fakeMapData) Elevations(_ context.Context, points []geo.Point) ([]float64, error) {
	elevations := make([]float64, len(points))
	for i, p := range points {
		elevations[i] = p.Lat * 1000
	}
	return elevations, nil
}

// Forecast returns clear, mild weather, with a thunderstorm in the second
// hour and darkness from 18:00 UTC.
func (f *fakeMapData) Forecast(_ context.Context, queries []mapdata.ForecastQuery) ([]mapdata.Weather, error) {
	f.mu.Lock()
	f.forecasts = append(f.forecasts, queries...)
	f.mu.Unlock()
	weather := make([]mapdata.Weather, len(queries))
	for i, q := range queries {
		hour := q.Time.UTC().Truncate(time.Hour)
		weather[i] = mapdata.Weather{
			Time:                hour,
			TemperatureCelsius:  10,
			ApparentTemperature: 8,
			WeatherCode:         1,
			FreezingLevelMeters: 3000,
			IsDay:               hour.Hour() < 18,
		}
		if hour.Equal(queries[0].Time.UTC().Truncate(time.Hour).Add(time.Hour)) {
			weather[i].WeatherCode = 95
		}
	}
	return weather, nil
}

func (f *fakeMapData) Search(_ context.Context, query string, limit int) ([]mapdata.Place, error) {
	return []mapdata.Place{{Name: query, Lat: 1, Lng: 2, Category: "natural", Type: "peak", BoundingBox: []float64{0.9, 1.1, 1.9, 2.1}}}, nil
}

func (f *fakeMapData) FindTrails(_ context.Context, center geo.Point, radius float64) ([]mapdata.Element, error) {
	way := mapdata.Element{Type: "way", ID: 42, Tags: map[string]string{"highway": "path", "name": "Ridge Trail", "surface": "dirt"}}
	for i := range 5 {
		way.Geometry = append(way.Geometry, struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lon"`
		}{Lat: center.Lat + float64(i)*0.001, Lng: center.Lng})
	}
	return []mapdata.Element{way}, nil
}

func (f *fakeMapData) FindFeatures(_ context.Context, center geo.Point, radius float64, filters []string) ([]mapdata.Element, error) {
	return []mapdata.Element{
		{Type: "node", ID: 1, Lat: center.Lat + 0.01, Lng: center.Lng, Tags: map[string]string{"natural": "peak", "name": "Far Peak", "ele": "1234"}},
		{Type: "node", ID: 2, Lat: center.Lat + 0.001, Lng: center.Lng, Tags: map[string]string{"amenity": "drinking_water"}},
	}, nil
}

// fakeLiveTrack returns a fixed position for a session.
type fakeLiveTrack struct {
	mu    sync.Mutex
	track livetrack.Track
}

func (f *fakeLiveTrack) Track(_ context.Context, s livetrack.Session) (livetrack.Track, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s.ID != "aaaaaaaa-bbbb" || s.Token != "TOKEN" {
		return livetrack.Track{}, fmt.Errorf("the LiveTrack session was not found; check the link")
	}
	return f.track, nil
}

func approx(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}
