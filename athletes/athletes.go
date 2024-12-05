package athletes

import (
	"log/slog"

	"github.com/catzkorn/trail-tools/store"
)

// Directory allows storing and querying of athelete and related data.
type Directory struct {
	log *slog.Logger
	Querier
}

// NewDirectory creates a new Directory from the provided logger and database.
func NewDirectory(log *slog.Logger, db *store.DB) (*Directory, error) {
	return &Directory{
		log:     log,
		Querier: &Queries{db: db},
	}, nil
}
