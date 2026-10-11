// Package services_test contains integration tests of the API services,
// running against a real database through the HTTP handlers and
// authentication middleware.
package services_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/gen/activities/v1/activitiesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/routing/v1/routingv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/strava/v1/stravav1connect"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/services/activity"
	"github.com/catzkorn/trail-tools/internal/services/auth"
	mapdatasvc "github.com/catzkorn/trail-tools/internal/services/mapdata"
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
	activities *activities.Repository
	syncer     *stravasvc.Syncer
	strava     *fakeStrava

	users      usersv1connect.UserServiceClient
	activity   activitiesv1connect.ActivityServiceClient
	stravaAPI  stravav1connect.StravaServiceClient
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
	e := &env{t: t, db: db, activities: activityRepo, strava: newFakeStrava()}
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

	mux := http.NewServeMux()
	server := connect.NewServer()
	usersv1connect.RegisterUserServiceHandler(server, userService)
	authv1connect.RegisterAuthServiceHandler(server, authService)
	activitiesv1connect.RegisterActivityServiceHandler(server, activityService)
	stravav1connect.RegisterStravaServiceHandler(server, stravaService)
	connecthttp.Mount(mux, server)

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
