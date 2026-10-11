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
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/athletes"
	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/gen/activities/v1/activitiesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/athletes/v1/athletesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/html"
	"github.com/catzkorn/trail-tools/internal/services/activity"
	"github.com/catzkorn/trail-tools/internal/services/athlete"
	"github.com/catzkorn/trail-tools/internal/services/auth"
	"github.com/catzkorn/trail-tools/internal/services/user"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/catzkorn/trail-tools/web"
	"github.com/johanbrandhorst/reload"
	"gitlab.com/greyxor/slogor"
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
	serveDir = flag.String(
		"serve-dir",
		"",
		"Optionally, the directory to serve as the root of the web application."+
			" If unset, will serve the compiled web application from the web package",
	)
	adminBootstrapToken = flag.String(
		"admin-bootstrap-token",
		"",
		"A secret that can be used as an invitation to create the first administrator,"+
			" at /invite/<token>, while there are no administrators",
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
		*serveDir,
		*adminBootstrapToken,
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
	serveDir string,
	adminBootstrapToken string,
) error {
	switch {
	case databaseUrl == "":
		return errors.New("database-url is required")
	case hostname == "":
		return errors.New("address is required")
	case port == "":
		return errors.New("port is required")
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
	activityRepo, err := activities.NewRepository(log, db)
	if err != nil {
		return fmt.Errorf("failed to create activity repository: %w", err)
	}

	userService := user.NewService(log, users)
	authService, err := auth.NewService(log, users, baseURL, adminBootstrapToken)
	if err != nil {
		return fmt.Errorf("failed to create auth service: %w", err)
	}
	activityService := activity.NewService(log, activityRepo)

	// Register all services on a single Connect server, served by the API mux
	server := connect.NewServer()
	athletesv1connect.RegisterAthleteServiceHandler(server, athlete.NewService(log, athletes))
	usersv1connect.RegisterUserServiceHandler(server, userService)
	authv1connect.RegisterAuthServiceHandler(server, authService)
	activitiesv1connect.RegisterActivityServiceHandler(server, activityService)
	apiMux := http.NewServeMux()
	connecthttp.Mount(apiMux, server)

	// Wrap the API mux in the authn middleware. Each service checks
	// authentication itself.
	authHandler, err := authn.NewAuthnMiddleware(ctx, log, users, apiMux)
	if err != nil {
		return fmt.Errorf("failed to create authentication middleware: %w", err)
	}

	// Create the core mux. This contains both the Connect API and HTML handlers.
	// Only the Connect API endpoints (registered to apiMux) are authenticated.
	mux := http.NewServeMux()
	// Serve the Connect API handlers through the authn middleware
	for _, path := range servicePaths(server) {
		mux.Handle(path, authHandler)
	}

	// Register the logout handler directly on the mux
	authn.RegisterLogoutHandler(log, mux, users)

	// Create the FS used to serve the frontend.
	webFs, _ := fs.Sub(web.Dist, "dist")
	if serveDir != "" {
		webFs = os.DirFS(serveDir)
	}
	fileServer := http.FileServerFS(webFs)

	// Serve index.js, index.js.map, index.css, favicon.ico and img/ directly
	mux.Handle("/index.js", fileServer)
	mux.Handle("/index.js.map", fileServer)
	mux.Handle("/index.css", fileServer)
	mux.Handle("/favicon.svg", fileServer)
	mux.Handle("/img/", fileServer)
	// For all other requests, serve the contents of index.html
	mux.Handle("/", html.ServeIndexHTML(serveDir != ""))

	// Add auto-reloading middleware if serving from a directory
	var srvHandler http.Handler = mux
	if serveDir != "" {
		srvHandler, err = reload.NewMiddleware(srvHandler, serveDir, log)
		if err != nil {
			return fmt.Errorf("failed to create reload middleware: %w", err)
		}
	}

	// Serve HTTP/2 without TLS (h2c) alongside HTTP/1.1.
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)

	srv := &http.Server{
		Addr:      address,
		Handler:   srvHandler,
		Protocols: protocols,
	}
	serveFn := srv.ListenAndServe
	if cert.Certificate != nil {
		// Over TLS, negotiate HTTP/2 via ALPN instead of h2c.
		protocols.SetHTTP2(true)
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

// servicePaths returns the URL path prefixes of the services registered on
// server, e.g. /users.v1.UserService/.
func servicePaths(server *connect.Server) []string {
	var paths []string
	for spec := range server.Specs() {
		path := spec.Procedure[:strings.LastIndexByte(spec.Procedure, '/')+1]
		if !slices.Contains(paths, path) {
			paths = append(paths, path)
		}
	}
	return paths
}
