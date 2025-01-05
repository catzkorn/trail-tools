package users

import (
	"log/slog"

	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/users/internal"
)

// Repository allows storing and querying of users and related data.
type Repository struct {
	log     *slog.Logger
	querier *internal.Queries
	db      *store.DB
}

// NewRepository creates a new Repository from the provided logger and database.
func NewRepository(log *slog.Logger, db *store.DB) (*Repository, error) {
	return &Repository{
		log:     log,
		querier: internal.New(db),
		db:      db,
	}, nil
}
