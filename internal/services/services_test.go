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
	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/services/auth"
	"github.com/catzkorn/trail-tools/internal/services/user"
	"github.com/catzkorn/trail-tools/internal/store/storetest"
	"github.com/catzkorn/trail-tools/internal/users"
)

// env is a running server for a test.
type env struct {
	srv *httptest.Server
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

	userService := user.NewService(log)
	authService, err := auth.NewService(log, userRepo, "http://localhost:8080", bootstrapToken)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	server := connect.NewServer()
	usersv1connect.RegisterUserServiceHandler(server, userService)
	authv1connect.RegisterAuthServiceHandler(server, authService)
	connecthttp.Mount(mux, server)

	handler, err := authn.NewAuthnMiddleware(ctx, log, userRepo, mux)
	if err != nil {
		t.Fatal(err)
	}
	e.srv = httptest.NewServer(handler)
	t.Cleanup(e.srv.Close)
	return e
}

// bootstrapToken is the admin bootstrap token in tests.
const bootstrapToken = "bootstrap-secret"

// wantCode fails the test unless err is a connect error with the code.
func wantCode(t *testing.T, err error, code connect.Code) {
	t.Helper()
	if connect.CodeOf(err) != code {
		t.Fatalf("got error %v, want code %v", err, code)
	}
}
