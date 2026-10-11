package routing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ImportState is the state of an import.
type ImportState string

const (
	ImportQueued  ImportState = "queued"
	ImportRunning ImportState = "running"
	ImportDone    ImportState = "done"
	ImportFailed  ImportState = "failed"
)

// ImportRecord is an import of an area of OpenStreetMap data into the routing
// graph.
type ImportRecord struct {
	ID         pgtype.UUID
	Region     string
	Bounds     geo.Bounds
	Extract    Extract
	State      ImportState
	Message    string
	EdgeCount  int
	CreateTime time.Time
	StartTime  pgtype.Timestamptz
	FinishTime pgtype.Timestamptz
}

// Region is an imported area of the routing graph.
type Region struct {
	Name      string
	Bounds    geo.Bounds
	EdgeCount int
	// The space the region takes, estimated from its share of the edges
	// table and its indexes.
	SizeBytes  int64
	ImportTime time.Time
}

const importColumns = `id, region, south, west, north, east, extract_name, extract_url,
	state::text, message, edge_count, create_time, start_time, finish_time`

func scanImport(row pgx.Row) (*ImportRecord, error) {
	var r ImportRecord
	var state string
	if err := row.Scan(&r.ID, &r.Region, &r.Bounds.MinLat, &r.Bounds.MinLng, &r.Bounds.MaxLat, &r.Bounds.MaxLng,
		&r.Extract.Name, &r.Extract.URL, &state, &r.Message, &r.EdgeCount, &r.CreateTime, &r.StartTime, &r.FinishTime); err != nil {
		return nil, err
	}
	r.State = ImportState(state)
	return &r, nil
}

// CreateImport records a new import, queued to run.
func CreateImport(ctx context.Context, db *store.DB, region string, bounds geo.Bounds, extract Extract, createdBy pgtype.UUID) (*ImportRecord, error) {
	r, err := scanImport(db.QueryRow(ctx, `insert into routing_imports (
		region, south, west, north, east, extract_name, extract_url, created_by
	) values ($1, $2, $3, $4, $5, $6, $7, $8)
	returning `+importColumns,
		region, bounds.MinLat, bounds.MinLng, bounds.MaxLat, bounds.MaxLng, extract.Name, extract.URL, createdBy))
	if err != nil {
		return nil, fmt.Errorf("failed to create the import: %w", err)
	}
	return r, nil
}

// GetImport gets an import.
func GetImport(ctx context.Context, db *store.DB, id pgtype.UUID) (*ImportRecord, error) {
	r, err := scanImport(db.QueryRow(ctx, `select `+importColumns+` from routing_imports where id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get the import: %w", err)
	}
	return r, nil
}

// ListImports lists the most recent imports, newest first.
func ListImports(ctx context.Context, db *store.DB, limit int) ([]*ImportRecord, error) {
	rows, err := db.Query(ctx, `select `+importColumns+` from routing_imports order by create_time desc limit $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list imports: %w", err)
	}
	defer rows.Close()
	var imports []*ImportRecord
	for rows.Next() {
		r, err := scanImport(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to list imports: %w", err)
		}
		imports = append(imports, r)
	}
	return imports, rows.Err()
}

// HasActiveImport reports whether an import is queued or running.
func HasActiveImport(ctx context.Context, db *store.DB) (bool, error) {
	var active bool
	if err := db.QueryRow(ctx, `select exists(select 1 from routing_imports where state in ('queued', 'running'))`).Scan(&active); err != nil {
		return false, fmt.Errorf("failed to check for imports: %w", err)
	}
	return active, nil
}

// FailImport marks an import as failed.
func FailImport(ctx context.Context, db *store.DB, id pgtype.UUID, message string) error {
	if _, err := db.Exec(ctx, `update routing_imports set state = 'failed', message = $2,
		finish_time = current_timestamp where id = $1`, id, message); err != nil {
		return fmt.Errorf("failed to update the import: %w", err)
	}
	return nil
}

// RunImport runs a queued import, recording its progress. It is called by
// the server for imports in the background, and by the import-osm command
// for imports run as a job.
func RunImport(ctx context.Context, log *slog.Logger, db *store.DB, dem *DEM, id pgtype.UUID) error {
	r, err := GetImport(ctx, db, id)
	if err != nil {
		return err
	}
	if _, err := db.Exec(ctx, `update routing_imports set state = 'running', message = '',
		start_time = current_timestamp where id = $1`, id); err != nil {
		return fmt.Errorf("failed to update the import: %w", err)
	}
	count, importErr := Import(ctx, log, db, ImportConfig{
		Region: r.Region,
		Source: r.Extract.URL,
		Bounds: r.Bounds,
		DEM:    dem,
	})
	// Record the result even if ctx was cancelled.
	ctx = context.WithoutCancel(ctx)
	if importErr != nil {
		if err := FailImport(ctx, db, id, importErr.Error()); err != nil {
			log.ErrorContext(ctx, "Failed to record the failed import", slog.Any("error", err))
		}
		return importErr
	}
	if _, err := db.Exec(ctx, `update routing_imports set state = 'done', edge_count = $2,
		finish_time = current_timestamp where id = $1`, id, count); err != nil {
		return fmt.Errorf("failed to update the import: %w", err)
	}
	return nil
}

// ListRegions lists the imported regions, by name.
func ListRegions(ctx context.Context, db *store.DB) ([]Region, error) {
	rows, err := db.Query(ctx, `select name, ST_YMin(bounds), ST_XMin(bounds), ST_YMax(bounds), ST_XMax(bounds),
		edge_count, import_time from routing_regions order by name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}
	defer rows.Close()
	var regions []Region
	for rows.Next() {
		var r Region
		if err := rows.Scan(&r.Name, &r.Bounds.MinLat, &r.Bounds.MinLng, &r.Bounds.MaxLat, &r.Bounds.MaxLng, &r.EdgeCount, &r.ImportTime); err != nil {
			return nil, fmt.Errorf("failed to list regions: %w", err)
		}
		regions = append(regions, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}
	var edgesSize int64
	if err := db.QueryRow(ctx, `select pg_total_relation_size('routing_edges')`).Scan(&edgesSize); err != nil {
		return nil, fmt.Errorf("failed to get the routing graph's size: %w", err)
	}
	total := 0
	for _, r := range regions {
		total += r.EdgeCount
	}
	if total > 0 {
		for i := range regions {
			regions[i].SizeBytes = edgesSize * int64(regions[i].EdgeCount) / int64(total)
		}
	}
	return regions, nil
}

// DeleteRegion deletes a region and its edges from the routing graph.
func DeleteRegion(ctx context.Context, db *store.DB, name string) error {
	tag, err := db.Exec(ctx, `delete from routing_regions where name = $1`, name)
	if err != nil {
		return fmt.Errorf("failed to delete the region: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// DatabaseSize returns the size of the database in bytes.
func DatabaseSize(ctx context.Context, db *store.DB) (int64, error) {
	var size int64
	if err := db.QueryRow(ctx, `select pg_database_size(current_database())`).Scan(&size); err != nil {
		return 0, fmt.Errorf("failed to get the database size: %w", err)
	}
	return size, nil
}
