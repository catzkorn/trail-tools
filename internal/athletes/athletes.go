package athletes

import (
	"log/slog"

	"github.com/catzkorn/trail-tools/internal/athletes/internal/gen"
	"github.com/catzkorn/trail-tools/internal/store"
)

// Repository allows storing and querying of atheletes and related data.
type Repository struct {
	log *slog.Logger
	q   *gen.Queries
}

// NewRepository creates a new Repository from the provided logger and database.
func NewRepository(log *slog.Logger, db *store.DB) (*Repository, error) {
	return &Repository{
		log: log,
		q:   gen.New(db),
	}, nil
}
