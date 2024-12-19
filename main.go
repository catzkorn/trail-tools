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
	"time"

	"github.com/catzkorn/trail-tools/athletes"
	"github.com/catzkorn/trail-tools/gen/athletes/v1/athletesv1connect"
	"github.com/catzkorn/trail-tools/oidc"
	"github.com/catzkorn/trail-tools/services/athlete"
	"github.com/catzkorn/trail-tools/store"
	"github.com/catzkorn/trail-tools/web"
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
)

func main() {
	flag.Parse()
	log := slog.New(slogor.NewHandler(os.Stdout, slogor.SetTimeFormat(time.Stamp)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, log, *databaseUrl, *address, *oidcClientID, *oidcClientSecret, *oidcIssuerURL); err != nil {
		log.Error("Failed running app", slogor.Err(err))
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
	dir, err := athletes.NewDirectory(log, db)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	apiPath, apiHandler := athletesv1connect.NewAthleteServiceHandler(athlete.NewService(log, dir))
	authAPI, err := oidc.NewAuthnMiddleware(ctx, log, oidcIssuerURL, oidcClientID, apiHandler)
	if err != nil {
		return fmt.Errorf("failed to create authn interceptor: %w", err)
	}
	mux := http.NewServeMux()
	// Serve the Connect API
	mux.Handle(apiPath, authAPI)
	// Serve OIDC handlers
	if err := oidc.RegisterHandlers(ctx, log, "http://"+address, oidcClientID, oidcClientSecret, oidcIssuerURL, mux); err != nil {
		return fmt.Errorf("failed to register OIDC handlers: %w", err)
	}
	// Serve index.js, index.js.map and index.css directly
	webFs, _ := fs.Sub(web.Dist, "dist")
	mux.Handle("/index.js", http.FileServer(http.FS(webFs)))
	mux.Handle("/index.js.map", http.FileServer(http.FS(webFs)))
	mux.Handle("/index.css", http.FileServer(http.FS(webFs)))
	// For all other requests, serve the contents of index.html
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, webFs, "index.html")
	}))

	srv := &http.Server{
		Addr: address,
		// Use h2c so we can serve HTTP/2 without TLS.
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	go func() {
		<-ctx.Done()
		log.Info("Interrupt received, shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("Failed to shutdown server", slogor.Err(err))
		}
	}()
	log.Info("Serving on", slog.String("addr", "localhost:8080"))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}
