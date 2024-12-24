package users

import (
	"log/slog"

	"github.com/catzkorn/trail-tools/store"
)

// Repository allows storing and querying of users and related data.
type Repository struct {
	log *slog.Logger
	Querier
}

// NewRepository creates a new Repository from the provided logger and database.
func NewRepository(log *slog.Logger, db *store.DB) (*Repository, error) {
	return &Repository{
		log:     log,
		Querier: &Queries{db: db},
	}, nil
}
