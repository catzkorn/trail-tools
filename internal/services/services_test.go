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

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/services/auth"
	"github.com/catzkorn/trail-tools/internal/services/user"
	"github.com/catzkorn/trail-tools/internal/store/storetest"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/jackc/pgx/v5/pgtype"
)

// env is a running server for a test, with clients authenticated as a user.
type env struct {
	userID pgtype.UUID
	token  string
	srv    *httptest.Server

	users usersv1connect.UserServiceClient
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
	e := &env{}
	e.userID = storetest.NewUser(t, db)
	_, e.token, err = userRepo.CreateAPIToken(ctx, e.userID, "test")
	if err != nil {
		t.Fatal(err)
	}

	userService := user.NewService(log, userRepo)
	authService, err := auth.NewService(log, userRepo, "http://localhost:8080", bootstrapToken)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle(usersv1connect.NewUserServiceHandler(userService))
	mux.Handle(authv1connect.NewAuthServiceHandler(authService))
	handler, err := authn.NewAuthnMiddleware(ctx, log, userRepo, mux)
	if err != nil {
		t.Fatal(err)
	}
	e.srv = httptest.NewServer(handler)
	t.Cleanup(e.srv.Close)

	client := e.srv.Client()
	opts := connect.WithInterceptors(bearerInterceptor(e.token))
	e.users = usersv1connect.NewUserServiceClient(client, e.srv.URL, opts)
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
