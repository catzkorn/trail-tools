// Store is the package that manages the database connection and migrations.
// It is used by other packages that need access to the database.
package store

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"net/url"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	decimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	slogadapter "github.com/mcosta74/pgx-slog"
)

//go:embed migrations/*.sql
var fs embed.FS

type DB struct {
	*pgx.Conn
}

// New creates a new Directory, connecting it to the postgres server on
// the URL provided.
func New(ctx context.Context, logger *slog.Logger, pgURL *url.URL) (*DB, error) {
	connURL := *pgURL
	c, err := pgx.ParseConfig(connURL.String())
	if err != nil {
		return nil, fmt.Errorf("parsing postgres URI: %w", err)
	}

	c.Tracer = &tracelog.TraceLog{
		Logger:   slogadapter.NewLogger(logger),
		LogLevel: tracelog.LogLevelTrace,
	}
	conn, err := pgx.ConnectConfig(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	err = validateSchema(conn, pgURL.Scheme)
	if err != nil {
		return nil, fmt.Errorf("validating schema: %w", err)
	}
	// Register decimal type
	decimal.Register(conn.TypeMap())

	return &DB{Conn: conn}, nil
}

// Migrate migrates the Postgres schema to the current version.
func validateSchema(conn *pgx.Conn, scheme string) error {
	sourceInstance, err := iofs.New(fs, "migrations")
	if err != nil {
		return err
	}
	db := stdlib.OpenDB(*conn.Config())
	defer db.Close()
	var driverInstance database.Driver
	switch scheme {
	case "postgres", "postgresql":
		driverInstance, err = pgxmigrate.WithInstance(db, new(pgxmigrate.Config))
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
