package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/catzkorn/trail-tools/athletes"
	"github.com/catzkorn/trail-tools/gen/athletes/v1/athletesv1connect"
	"github.com/catzkorn/trail-tools/services/athlete"
	"github.com/catzkorn/trail-tools/store"
	"gitlab.com/greyxor/slogor"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var databaseUrl = flag.String(
	"database-url",
	"",
	"The URL of the postgres database in which to store the state of the application",
)

func main() {
	flag.Parse()
	log := slog.New(slogor.NewHandler(os.Stdout, slogor.SetTimeFormat(time.Stamp)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, log, *databaseUrl); err != nil {
		log.Error("Failed running app", slogor.Err(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, log *slog.Logger, databaseUrl string) error {
	if databaseUrl == "" {
		return errors.New("database-url is required")
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
	mux := http.NewServeMux()
	mux.Handle(athletesv1connect.NewAthleteServiceHandler(athlete.NewService(log, dir)))
	srv := &http.Server{
		Addr: "localhost:8080",
		// Use h2c so we can serve HTTP/2 without TLS.
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}
	go func() {
		<-ctx.Done()
		log.Info("Shutting down")
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
