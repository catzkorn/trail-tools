package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/catzkorn/trail-tools/internal/athletes"
	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/gen/athletes/v1/athletesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/oidc"
	"github.com/catzkorn/trail-tools/internal/services/athlete"
	"github.com/catzkorn/trail-tools/internal/services/user"
	"github.com/catzkorn/trail-tools/internal/services/webauthn"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/catzkorn/trail-tools/web"
	wauthn "github.com/go-webauthn/webauthn/webauthn"
	"gitlab.com/greyxor/slogor"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

const (
	defaultHostname = "localhost"
	defaultPort     = "8080"
)

var (
	logLevel = flag.String(
		"log-level",
		"info",
		"The log level to use",
	)
	databaseUrl = flag.String(
		"database-url",
		"",
		"The URL of the postgres database in which to store the state of the application",
	)
	hostname = flag.String(
		"hostname",
		defaultHostname,
		"The hostname on which to serve the application",
	)
	port = flag.String(
		"port",
		defaultPort,
		"The port on which to serve the application",
	)
	tlsKeyPath = flag.String(
		"tls-key",
		"",
		"The path to the private key used for serving the application over HTTPS",
	)
	tlsCertPath = flag.String(
		"tls-cert",
		"",
		"The path to the certificate used for serving the application over HTTPS",
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
)

func main() {
	flag.Parse()
	var slogLevel slog.Level
	if err := slogLevel.UnmarshalText([]byte(*logLevel)); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse log level: %v\n", err)
		os.Exit(1)
	}
	logger := slog.New(slogor.NewHandler(
		os.Stdout,
		slogor.SetTimeFormat(time.Stamp),
		slogor.SetLevel(slogLevel),
	))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(
		ctx,
		logger,
		*databaseUrl,
		*hostname,
		*port,
		*tlsKeyPath,
		*tlsCertPath,
		*oidcClientID,
		*oidcClientSecret,
		*oidcIssuerURL,
		*serveDir,
	); err != nil {
		logger.ErrorContext(ctx, "Failed running app", slogor.Err(err))
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	log *slog.Logger,
	databaseUrl string,
	hostname string,
	port string,
	tlsKeyPath string,
	tlsCertPath string,
	oidcClientID string,
	oidcClientSecret string,
	oidcIssuerURL string,
	serveDir string,
) error {
	switch {
	case databaseUrl == "":
		return errors.New("database-url is required")
	case hostname == "":
		return errors.New("address is required")
	case port == "":
		return errors.New("port is required")
	case oidcClientID == "":
		return errors.New("oidc-client-id is required")
	case oidcClientSecret == "":
		return errors.New("oidc-client-secret is required")
	case oidcIssuerURL == "":
		return errors.New("oidc-issuer-url is required")
	case tlsKeyPath != "" && tlsCertPath == "" || tlsKeyPath == "" && tlsCertPath != "":
		return errors.New("tls-cert and tls-key must be set together")
	}
	dbUrl, err := url.Parse(databaseUrl)
	if err != nil {
		return fmt.Errorf("failed to parse database-url: %w", err)
	}
	address := net.JoinHostPort(hostname, port)
	baseURL := "http://" + address
	var cert tls.Certificate
	if tlsCertPath != "" && tlsKeyPath != "" {
		cert, err = tls.LoadX509KeyPair(tlsCertPath, tlsKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load TLS certificate: %w", err)
		}
		baseURL = "https://" + address
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
	athletePath, athleteHandler := athletesv1connect.NewAthleteServiceHandler(athlete.NewService(log, athletes))
	usersPath, usersHandler := usersv1connect.NewUserServiceHandler(user.NewService(log))
	apiMux := http.NewServeMux()
	apiMux.Handle(athletePath, athleteHandler)
	apiMux.Handle(usersPath, usersHandler)

	// Wrap the API mux in the OIDC authn middleware
	// All API calls will require authentication.
	oidcHandler, err := oidc.NewOIDCMiddleware(ctx, log, oidcIssuerURL, oidcClientID, apiMux)
	if err != nil {
		return fmt.Errorf("failed to create OIDC middleware: %w", err)
	}
	authHandler, err := authn.NewAuthnMiddleware(ctx, log, users, oidcHandler)
	if err != nil {
		return fmt.Errorf("failed to create authentication middleware: %w", err)
	}

	// Create the core mux. This contains both the Connect API, OIDC, WebAuthn and HTML handlers.
	// Only the Connect API endpoints (registered to apiMux) are authenticated.
	mux := http.NewServeMux()
	// Serve the Connect API handlers through the authn middleware
	mux.Handle(athletePath, authHandler)
	mux.Handle(usersPath, authHandler)

	// Serve OIDC handlers directly on the mux (unauthenticated)
	oidcLogout, err := oidc.RegisterHandlers(ctx, log, baseURL, oidcClientID, oidcClientSecret, oidcIssuerURL, users, mux)
	if err != nil {
		return fmt.Errorf("failed to register OIDC handlers: %w", err)
	}

	// Register authn and oidc logout handler directly on the mux
	authn.RegisterLogoutHandler(log, mux, users, oidcLogout)

	webAuthn, err := wauthn.New(&wauthn.Config{
		RPDisplayName: "Trail tools",
		RPID:          hostname,
		RPOrigins:     []string{baseURL},
	})
	if err != nil {
		return fmt.Errorf("failed to create webauthn: %w", err)
	}
	// Serve the WebAuthn handlers directly on the mux (unauthenticated)
	webauthn.RegisterHandlers(mux, log, webAuthn, users)

	// Create the FS used to serve the frontend.
	webFs, _ := fs.Sub(web.Dist, "dist")
	if serveDir != "" {
		webFs = os.DirFS(serveDir)
	}
	fileServer := http.FileServer(http.FS(webFs))

	// Serve index.js, index.js.map, index.css and favicon.ico directly
	mux.Handle("/index.js", fileServer)
	mux.Handle("/index.js.map", fileServer)
	mux.Handle("/index.css", fileServer)
	mux.Handle("/favicon.svg", fileServer)
	// For all other requests, serve the contents of index.html
	mux.Handle("/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, webFs, "index.html")
		}),
	)

	srv := &http.Server{
		Addr: address,
		// Use h2c so we can serve HTTP/2 without TLS.
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	serveFn := srv.ListenAndServe
	if cert.Certificate != nil {
		srv.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		serveFn = func() error {
			return srv.ListenAndServeTLS("", "")
		}
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
	log.InfoContext(ctx, "Serving on", slog.String("addr", address))
	if err := serveFn(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}
