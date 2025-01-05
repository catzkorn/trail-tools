package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/catzkorn/trail-tools/athletes"
	"github.com/catzkorn/trail-tools/gen/athletes/v1/athletesv1connect"
	"github.com/catzkorn/trail-tools/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/oidc"
	"github.com/catzkorn/trail-tools/services/athlete"
	"github.com/catzkorn/trail-tools/services/user"
	"github.com/catzkorn/trail-tools/services/webauthn"
	"github.com/catzkorn/trail-tools/store"
	"github.com/catzkorn/trail-tools/users"
	"github.com/catzkorn/trail-tools/web"
	"github.com/go-chi/chi/v5/middleware"
	wauthn "github.com/go-webauthn/webauthn/webauthn"
	"gitlab.com/greyxor/slogor"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	databaseUrl = flag.String(
		"database-url",
		"",
		"The URL of the postgres database in which to store the state of the application",
	)
	address = flag.String(
		"address",
		"localhost:8080",
		"The address on which to serve the application",
	)
	oidcClientID = flag.String(
		"oidc-client-id",
		"",
		"The client ID to use when authenticating with the OIDC issuer",
	)
	oidcClientSecret = flag.String(
		"oidc-client-secret",
		"",
		"The client secret to use when authenticating with the OIDC issuer",
	)
	oidcIssuerURL = flag.String(
		"oidc-issuer-url",
		"https://accounts.google.com",
		"The URL of the OIDC issuer to use for authentication",
	)
	serveDir = flag.String(
		"serve-dir",
		"",
		"Optionally, the directory to serve as the root of the web application."+
			" If unset, will serve the compiled web application from the web package",
	)
	webAuthnRPID = flag.String(
		"webauthn-rp-id",
		"localhost",
		"The Relying Party ID for WebAuthn, defaults to localhost",
	)
	webAuthnRPOrigins = flag.String(
		"webauthn-rp-origins",
		"http://localhost",
		"Comma-separated list of Relying Party Origins for WebAuthn, defaults to http://localhost",
	)
)

func main() {
	flag.Parse()
	log := slog.New(slogor.NewHandler(os.Stdout, slogor.SetTimeFormat(time.Stamp)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(
		ctx,
		log,
		*databaseUrl,
		*address,
		*oidcClientID,
		*oidcClientSecret,
		*oidcIssuerURL,
		*serveDir,
		*webAuthnRPID,
		*webAuthnRPOrigins,
	); err != nil {
		log.ErrorContext(ctx, "Failed running app", slogor.Err(err))
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	log *slog.Logger,
	databaseUrl string,
	address string,
	oidcClientID string,
	oidcClientSecret string,
	oidcIssuerURL string,
	serveDir string,
	webAuthnRPID string,
	webAuthnRPOrigins string,
) error {
	switch {
	case databaseUrl == "":
		return errors.New("database-url is required")
	case address == "":
		return errors.New("address is required")
	case oidcClientID == "":
		return errors.New("oidc-client-id is required")
	case oidcClientSecret == "":
		return errors.New("oidc-client-secret is required")
	case oidcIssuerURL == "":
		return errors.New("oidc-issuer-url is required")
	}
	dbUrl, err := url.Parse(databaseUrl)
	if err != nil {
		return fmt.Errorf("failed to parse database-url: %w", err)
	}
	db, err := store.New(ctx, log, dbUrl)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	users, err := users.NewRepository(log, db)
	if err != nil {
		return fmt.Errorf("failed to create user directory: %w", err)
	}
	athletes, err := athletes.NewRepository(log, db)
	if err != nil {
		return fmt.Errorf("failed to create athlete directory: %w", err)
	}

	// Register all handlers to a single mux
	athletePath, athleteHandler := athletesv1connect.NewAthleteServiceHandler(athlete.NewService(log, users, athletes))
	usersPath, usersHandler := usersv1connect.NewUserServiceHandler(user.NewService(log, users))
	apiMux := http.NewServeMux()
	apiMux.Handle(athletePath, athleteHandler)
	apiMux.Handle(usersPath, usersHandler)

	// Wrap the mux in the OIDC authn middleware
	apiAuth, err := oidc.NewAuthnMiddleware(ctx, log, oidcIssuerURL, oidcClientID, apiMux)
	if err != nil {
		return fmt.Errorf("failed to create authn interceptor: %w", err)
	}

	mux := http.NewServeMux()
	// Serve the API handlers through the authn middleware
	mux.Handle(athletePath, apiAuth)
	mux.Handle(usersPath, apiAuth)
	// Serve OIDC handlers
	if err := oidc.RegisterHandlers(ctx, log, "http://"+address, oidcClientID, oidcClientSecret, oidcIssuerURL, users, mux); err != nil {
		return fmt.Errorf("failed to register OIDC handlers: %w", err)
	}
	// Serve WebAuthn handlers if configured
	if webAuthnRPID != "" && webAuthnRPOrigins != "" {
		wconfig := &wauthn.Config{
			RPDisplayName: "Trail tools",
			RPID:          webAuthnRPID,                          // Generally the FQDN for your site
			RPOrigins:     strings.Split(webAuthnRPOrigins, ","), // The origin URLs allowed for WebAuthn requests
		}

		webAuthn, err := wauthn.New(wconfig)
		if err != nil {
			return fmt.Errorf("failed to create webauthn: %w", err)
		}
		webauthn.Register(mux, log, webAuthn, users)
	}

	webFs, _ := fs.Sub(web.Dist, "dist")
	if serveDir != "" {
		webFs = os.DirFS(serveDir)
	}
	fileServer := middleware.NoCache(http.FileServer(http.FS(webFs)))

	// Serve index.js, index.js.map, index.css and favicon.ico directly
	mux.Handle("/index.js", fileServer)
	mux.Handle("/index.js.map", fileServer)
	mux.Handle("/index.css", fileServer)
	mux.Handle("/favicon.svg", fileServer)
	// For all other requests, serve the contents of index.html
	mux.Handle("/",
		middleware.NoCache(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.ServeFileFS(w, r, webFs, "index.html")
			}),
		),
	)

	srv := &http.Server{
		Addr: address,
		// Use h2c so we can serve HTTP/2 without TLS.
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	go func() {
		<-ctx.Done()
		log.InfoContext(ctx, "Interrupt received, shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.ErrorContext(ctx, "Failed to shutdown server", slogor.Err(err))
		}
	}()
	log.InfoContext(ctx, "Serving on", slog.String("addr", "localhost:8080"))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}
