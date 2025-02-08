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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jackc/pgx/v5/tracelog"
	slogadapter "github.com/mcosta74/pgx-slog"
)

//go:embed migrations/*.sql
var fs embed.FS

type DB struct {
	*pgxpool.Pool
}

// New creates a new Directory, connecting it to the postgres server on
// the URL provided.
func New(ctx context.Context, logger *slog.Logger, pgURL *url.URL) (*DB, error) {
	poolConf, err := pgxpool.ParseConfig((*pgURL).String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres URI: %w", err)
	}
	poolConf.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   &logLevelMapper{wrapped: slogadapter.NewLogger(logger)},
		LogLevel: tracelog.LogLevelTrace,
	}

	// Validate and/or run migrations
	err = validateSchema(*poolConf.ConnConfig, pgURL.Scheme)
	if err != nil {
		return nil, fmt.Errorf("failed to validate schema: %w", err)
	}

	// Automatically register our enum types on each new connection
	enumTypes, err := getEnumTypes(ctx, poolConf.ConnConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get enum types: %w", err)
	}
	poolConf.AfterConnect = registerEnums(enumTypes)

	pool, err := pgxpool.NewWithConfig(ctx, poolConf)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// validateSchema migrates the Postgres schema to the current version.
func validateSchema(conf pgx.ConnConfig, scheme string) error {
	sourceInstance, err := iofs.New(fs, "migrations")
	if err != nil {
		return err
	}
	db := stdlib.OpenDB(conf)
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

// getEnumTypes returns a list of all enum types in the database.
func getEnumTypes(ctx context.Context, conf *pgx.ConnConfig) ([]string, error) {
	conn, err := pgx.ConnectConfig(ctx, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	defer conn.Close(ctx)
	const queryEnums = `
   select t.typname as name 
     from pg_type t 
left join pg_catalog.pg_namespace n on n.oid = t.typnamespace 
    where (t.typrelid = 0 or (select c.relkind = 'c' from pg_catalog.pg_class c where c.oid = t.typrelid)) 
      and not exists(select 1 from pg_catalog.pg_type el where el.oid = t.typelem and el.typarray = t.oid)
      and n.nspname not in ('pg_catalog', 'information_schema');
`

	rows, err := conn.Query(ctx, queryEnums)
	if err != nil {
		return nil, fmt.Errorf("failed to query enums in postgres: %w", err)
	}

	var enums []string
	for rows.Next() {
		var enum string
		if err := rows.Scan(&enum); err != nil {
			return nil, fmt.Errorf("failed to scan enum: %w", err)
		}
		enums = append(enums, enum)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan enums in postgres: %w", err)
	}
	return enums, nil
}

// registerEnums returns a function that registers all enums in the database on each new connection.
func registerEnums(enums []string) func(ctx context.Context, conn *pgx.Conn) error {
	return func(ctx context.Context, conn *pgx.Conn) error {
		// Register all enums
		for _, enum := range enums {
			t, err := conn.LoadType(ctx, enum)
			if err != nil {
				return err
			}
			conn.TypeMap().RegisterType(t)

			ta, err := conn.LoadType(ctx, "_"+enum)
			if err != nil {
				return err
			}
			conn.TypeMap().RegisterType(ta)
		}

		// Register decimal type
		decimal.Register(conn.TypeMap())

		return nil
	}
}
