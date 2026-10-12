package routing

import (
	"context"
	"log/slog"
	"sync"

	"github.com/catzkorn/trail-tools/internal/elevation"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
)

// LocalRunner runs imports in the background of the server, one at a time,
// as an import can use a lot of memory. It suits servers that keep running
// between requests, such as a local server.
type LocalRunner struct {
	ctx context.Context
	log *slog.Logger
	db  *store.DB
	dem *elevation.DEM
	mu  sync.Mutex
	wg  sync.WaitGroup
}

// NewLocalRunner creates a new LocalRunner. Imports are cancelled when ctx
// is done, and recorded as failed, including those still waiting to run.
func NewLocalRunner(ctx context.Context, log *slog.Logger, db *store.DB, dem *elevation.DEM) *LocalRunner {
	return &LocalRunner{ctx: ctx, log: log, db: db, dem: dem}
}

// Run starts running an import in the background.
func (r *LocalRunner) Run(_ context.Context, id pgtype.UUID) error {
	r.wg.Go(func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if err := RunImport(r.ctx, r.log, r.db, r.dem, id); err != nil {
			r.log.ErrorContext(r.ctx, "Failed to import OpenStreetMap data", slog.Any("error", err))
		}
	})
	return nil
}

// Wait waits for running imports to finish.
func (r *LocalRunner) Wait() {
	r.wg.Wait()
}
