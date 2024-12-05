// Store is the package that manages the database connection and migrations.
// It is used by other packages that need access to the database.
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"net/url"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	slogadapter "github.com/mcosta74/pgx-slog"
)

//go:embed migrations/*.sql
var fs embed.FS

type DB struct {
	*sql.DB
}

// New creates a new Directory, connecting it to the postgres server on
// the URL provided.
func New(logger *slog.Logger, pgURL *url.URL) (*DB, error) {
	connURL := *pgURL
	c, err := pgx.ParseConfig(connURL.String())
	if err != nil {
		return nil, fmt.Errorf("parsing postgres URI: %w", err)
	}

	c.Tracer = &tracelog.TraceLog{
		Logger:   slogadapter.NewLogger(logger),
		LogLevel: tracelog.LogLevelTrace,
	}
	db := stdlib.OpenDB(*c)

	err = validateSchema(db, pgURL.Scheme)
	if err != nil {
		return nil, fmt.Errorf("validating schema: %w", err)
	}

	return &DB{DB: db}, nil
}

// Migrate migrates the Postgres schema to the current version.
func validateSchema(db *sql.DB, scheme string) error {
	sourceInstance, err := iofs.New(fs, "migrations")
	if err != nil {
		return err
	}
	var driverInstance database.Driver
	switch scheme {
	case "postgres", "postgresql":
		driverInstance, err = postgres.WithInstance(db, new(postgres.Config))
	default:
		return fmt.Errorf("unknown scheme: %q", scheme)
	}
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", sourceInstance, scheme, driverInstance)
	if err != nil {
		return err
	}
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}
	return sourceInstance.Close()
}
