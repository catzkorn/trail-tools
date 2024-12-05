package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"time"

	"github.com/catzkorn/trail-tools/athletes"
	"github.com/catzkorn/trail-tools/store"
	"gitlab.com/greyxor/slogor"
)

var databaseUrl = flag.String(
	"database-url",
	"",
	"The URL of the postgres database in which to store the state of the application",
)

func main() {
	flag.Parse()
	log := slog.New(slogor.NewHandler(os.Stdout, slogor.SetTimeFormat(time.Stamp)))
	if err := run(log, *databaseUrl); err != nil {
		log.Error("Failed running app", slogor.Err(err))
		os.Exit(1)
	}
}

func run(log *slog.Logger, databaseUrl string) error {
	if databaseUrl == "" {
		return errors.New("database-url is required")
	}
	dbUrl, err := url.Parse(databaseUrl)
	if err != nil {
		return fmt.Errorf("failed to parse database-url: %w", err)
	}
	db, err := store.New(log, dbUrl)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}
	dir, err := athletes.NewDirectory(log, db)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	_ = dir
	return nil
}
