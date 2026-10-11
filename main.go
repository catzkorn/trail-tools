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
	"path/filepath"
	"strings"
	"time"

	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/athletes"
	"github.com/catzkorn/trail-tools/internal/authn"
	"github.com/catzkorn/trail-tools/internal/gen/activities/v1/activitiesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/athletes/v1/athletesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/auth/v1/authv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/geo/v1/geov1connect"
	"github.com/catzkorn/trail-tools/internal/gen/oauth/v1/oauthv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/plans/v1/plansv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/races/v1/racesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/routes/v1/routesv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/routing/v1/routingv1connect"
	"github.com/catzkorn/trail-tools/internal/gen/strava/v1/stravav1connect"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/html"
	"github.com/catzkorn/trail-tools/internal/livetrack"
	"github.com/catzkorn/trail-tools/internal/mapdata"
	"github.com/catzkorn/trail-tools/internal/mcp"
	"github.com/catzkorn/trail-tools/internal/oauth"
	"github.com/catzkorn/trail-tools/internal/plans"
	"github.com/catzkorn/trail-tools/internal/races"
	"github.com/catzkorn/trail-tools/internal/routes"
	"github.com/catzkorn/trail-tools/internal/routing"
	"github.com/catzkorn/trail-tools/internal/services/activity"
	"github.com/catzkorn/trail-tools/internal/services/athlete"
	"github.com/catzkorn/trail-tools/internal/services/auth"
	geosvc "github.com/catzkorn/trail-tools/internal/services/geo"
	mapdatasvc "github.com/catzkorn/trail-tools/internal/services/mapdata"
	oauthsvc "github.com/catzkorn/trail-tools/internal/services/oauth"
	"github.com/catzkorn/trail-tools/internal/services/plan"
	"github.com/catzkorn/trail-tools/internal/services/race"
	"github.com/catzkorn/trail-tools/internal/services/raceplan"
	"github.com/catzkorn/trail-tools/internal/services/route"
	stravasvc "github.com/catzkorn/trail-tools/internal/services/strava"
	"github.com/catzkorn/trail-tools/internal/services/user"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/strava"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/catzkorn/trail-tools/web"
	"github.com/johanbrandhorst/reload"
	"gitlab.com/greyxor/slogor"
)

const (
	defaultHostname = "localhost"
	defaultPort     = "8080"
	// mcpPath is the path of the MCP server.
	mcpPath = "/mcp"
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
	stravaClientID = flag.String(
		"strava-client-id",
		"",
		"The client ID of the Strava API application. If unset, Strava sync is disabled",
	)
	stravaClientSecret = flag.String(
		"strava-client-secret",
		"",
		"The client secret of the Strava API application",
	)
	stravaSyncMode = flag.String(
		"strava-sync-mode",
		"background",
		"How Strava syncs run: \"background\" runs them in the server, \"scheduled\" runs them when"+
			" a scheduler calls "+stravasvc.SyncPath+", for servers that don't run between requests, like Cloud Run",
	)
	schedulerServiceAccount = flag.String(
		"scheduler-service-account",
		"",
		"The email of the service account allowed to call "+stravasvc.SyncPath+" with a Google signed ID token,"+
			" e.g. from Cloud Scheduler. Required with -strava-sync-mode=scheduled",
	)
	schedulerAudience = flag.String(
		"scheduler-audience",
		"trail-tools-scheduler",
		"The audience of the scheduler's ID tokens",
	)
	adminBootstrapToken = flag.String(
		"admin-bootstrap-token",
		"",
		"A secret that can be used as an invitation to create the first administrator,"+
			" at /invite/<token>, while there are no administrators",
	)
	overpassURLs = flag.String(
		"overpass-urls",
		strings.Join(mapdata.DefaultOverpassURLs, ","),
		"Comma separated Overpass API URLs used to query OpenStreetMap, tried in order",
	)
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "import-osm" {
		os.Exit(importOSM(os.Args[2:]))
	}
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
		*stravaClientID,
		*stravaClientSecret,
		*adminBootstrapToken,
		*stravaSyncMode,
		*schedulerServiceAccount,
		*schedulerAudience,
		mapdata.Config{
			OverpassURLs: strings.Split(*overpassURLs, ","),
		},
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
	stravaClientID string,
	stravaClientSecret string,
	adminBootstrapToken string,
	stravaSyncMode string,
	schedulerServiceAccount string,
	schedulerAudience string,
	mapConfig mapdata.Config,
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
	case stravaClientID != "" && stravaClientSecret == "":
		return errors.New("strava-client-secret is required when strava-client-id is set")
	case stravaSyncMode != "background" && stravaSyncMode != "scheduled":
		return errors.New(`strava-sync-mode must be "background" or "scheduled"`)
	case stravaSyncMode == "scheduled" && schedulerServiceAccount == "":
		return errors.New("scheduler-service-account is required with -strava-sync-mode=scheduled")
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
	routeRepo, err := routes.NewRepository(log, db)
	if err != nil {
		return fmt.Errorf("failed to create route repository: %w", err)
	}
	raceRepo, err := races.NewRepository(log, db)
	if err != nil {
		return fmt.Errorf("failed to create race repository: %w", err)
	}
	planRepo, err := plans.NewRepository(log, db)
	if err != nil {
		return fmt.Errorf("failed to create plan repository: %w", err)
	}
	mapData := mapdata.New(mapConfig)

	// Strava is optional, as it requires API credentials.
	var stravaSyncer *stravasvc.Syncer
	var stravaOAuth *stravasvc.OAuth
	if stravaClientID != "" {
		stravaClient := strava.New(strava.Config{
			ClientID:     stravaClientID,
			ClientSecret: stravaClientSecret,
			RedirectURL:  baseURL + stravasvc.CallbackPath,
		})
		background := stravaSyncMode == "background"
		stravaSyncer = stravasvc.NewSyncer(ctx, log, activityRepo, stravaClient, background)
		stravaOAuth = stravasvc.NewOAuth(log, stravaClient, activityRepo, stravaSyncer, []byte(stravaClientSecret))
		if background {
			go stravaSyncer.RunBackground(ctx)
			defer stravaSyncer.Wait()
		}
	} else {
		log.InfoContext(ctx, "Strava sync disabled, set -strava-client-id and -strava-client-secret to enable it")
	}

	userService := user.NewService(log, users)
	authService, err := auth.NewService(log, users, baseURL, adminBootstrapToken)
	if err != nil {
		return fmt.Errorf("failed to create auth service: %w", err)
	}
	activityService := activity.NewService(log, activityRepo)
	stravaService := stravasvc.NewService(log, activityRepo, stravaSyncer, stravaOAuth)
	routeService := route.NewService(log, routeRepo, activityRepo, routing.NewRouter(db), mapData)
	geoService := geosvc.NewService(log, mapData, activityRepo)
	importRunner := routing.NewLocalRunner(ctx, log, db, routing.NewDEM(routing.DefaultDEMURL, filepath.Join(os.TempDir(), "trail-tools-dem")))
	defer importRunner.Wait()
	mapDataService := mapdatasvc.NewService(log, db, routing.NewExtracts(routing.DefaultGeofabrikIndexURL), importRunner)
	planService := plan.NewService(log, planRepo)
	raceService := race.NewService(log, raceRepo, routeRepo)
	racePlanService := raceplan.NewService(log, raceRepo, routeRepo, livetrack.New("", nil), baseURL)
	// MCP clients that require OAuth, like claude.ai, authorize with this
	// authorization server instead of API tokens.
	oauthServer := oauth.NewServer(log, users, baseURL, mcpPath)
	oauthService := oauthsvc.NewService(log, oauthServer, users)

	// Register all handlers to a single mux
	apiMux := http.NewServeMux()
	var apiPaths []string
	for _, register := range []func() (string, http.Handler){
		func() (string, http.Handler) {
			return athletesv1connect.NewAthleteServiceHandler(athlete.NewService(log, athletes))
		},
		func() (string, http.Handler) { return usersv1connect.NewUserServiceHandler(userService) },
		func() (string, http.Handler) { return authv1connect.NewAuthServiceHandler(authService) },
		func() (string, http.Handler) { return activitiesv1connect.NewActivityServiceHandler(activityService) },
		func() (string, http.Handler) { return stravav1connect.NewStravaServiceHandler(stravaService) },
		func() (string, http.Handler) { return routesv1connect.NewRouteServiceHandler(routeService) },
		func() (string, http.Handler) { return geov1connect.NewGeoServiceHandler(geoService) },
		func() (string, http.Handler) { return routingv1connect.NewMapDataServiceHandler(mapDataService) },
		func() (string, http.Handler) { return plansv1connect.NewTrainingPlanServiceHandler(planService) },
		func() (string, http.Handler) { return racesv1connect.NewRaceServiceHandler(raceService) },
		func() (string, http.Handler) { return racesv1connect.NewRacePlanServiceHandler(racePlanService) },
		func() (string, http.Handler) { return oauthv1connect.NewOAuthServiceHandler(oauthService) },
	} {
		path, handler := register()
		apiMux.Handle(path, handler)
		apiPaths = append(apiPaths, path)
	}

	// Wrap the API mux in the authn middleware. Each service checks
	// authentication itself.
	authHandler, err := authn.NewAuthnMiddleware(ctx, log, users, apiMux)
	if err != nil {
		return fmt.Errorf("failed to create authentication middleware: %w", err)
	}
	authenticate := func(next http.Handler) http.Handler {
		// NewAuthnMiddleware never returns an error.
		h, _ := authn.NewAuthnMiddleware(ctx, log, users, next)
		return h
	}

	// Create the core mux. This contains both the Connect API and HTML handlers.
	// Only the Connect API endpoints (registered to apiMux) are authenticated.
	mux := http.NewServeMux()
	// Serve the Connect API handlers through the authn middleware
	for _, path := range apiPaths {
		mux.Handle(path, authHandler)
	}

	// Serve the same API to AI agents over MCP. MCP clients authenticate
	// with API tokens or OAuth access tokens.
	mcpServer, err := mcp.NewServer(log,
		mcp.Service{Name: usersv1connect.UserServiceName, Handler: userService},
		mcp.Service{Name: activitiesv1connect.ActivityServiceName, Handler: activityService},
		mcp.Service{Name: stravav1connect.StravaServiceName, Handler: stravaService},
		mcp.Service{Name: routesv1connect.RouteServiceName, Handler: routeService},
		mcp.Service{Name: geov1connect.GeoServiceName, Handler: geoService},
		mcp.Service{Name: plansv1connect.TrainingPlanServiceName, Handler: planService},
		mcp.Service{Name: racesv1connect.RaceServiceName, Handler: raceService},
		mcp.Service{Name: racesv1connect.RacePlanServiceName, Handler: racePlanService},
	)
	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}
	resourceMetadataURL := oauthServer.ResourceMetadataURL()
	mcpHandler, err := authn.NewAuthnMiddleware(ctx, log, users,
		authn.RequireUser(resourceMetadataURL, mcpServer.Handler()),
		authn.WithResourceMetadata(resourceMetadataURL),
	)
	if err != nil {
		return fmt.Errorf("failed to create MCP authentication middleware: %w", err)
	}
	mux.Handle(mcpPath, mcpHandler)

	// Serve Strava OAuth handlers
	if stravaOAuth != nil {
		stravaOAuth.RegisterHandlers(mux, authenticate)
	}
	// Serve the scheduled sync endpoint, only for the scheduler
	if stravaSyncer != nil && !stravaSyncer.Background() {
		syncHandler, err := authn.RequireGoogleServiceAccount(ctx, log, schedulerAudience, schedulerServiceAccount,
			stravasvc.SyncHandler(log, stravaSyncer))
		if err != nil {
			return fmt.Errorf("failed to create scheduled sync handler: %w", err)
		}
		mux.Handle(stravasvc.SyncPath, syncHandler)
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
	indexHandler := html.ServeIndexHTML(serveDir != "")
	mux.Handle("/", indexHandler)
	// Serve the OAuth endpoints, with the frontend's consent page for valid
	// authorization requests.
	oauthServer.RegisterHandlers(mux, indexHandler)
	// Serve routes as GPX files, falling back to the frontend for other
	// /routes/ pages.
	mux.Handle("/routes/", authenticate(routeService.GPXHandler(indexHandler)))
	// Registering /routes/ makes the mux redirect /routes to /routes/, so
	// serve the frontend for /routes explicitly.
	mux.Handle("/routes", indexHandler)

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
