package athletes

import (
	"context"
	"log/slog"

	"github.com/catzkorn/trail-tools/store"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

// Repository allows storing and querying of atheletes and related data.
type Repository struct {
	log *slog.Logger
	q   Querier
}

// NewRepository creates a new Repository from the provided logger and database.
func NewRepository(log *slog.Logger, db *store.DB) (*Repository, error) {
	return &Repository{
		log: log,
		q:   &Queries{db: db},
	}, nil
}

func (d *Repository) AddActivity(ctx context.Context, name string, athleteID pgtype.UUID) (Activity, error) {
	return d.q.AddActivity(ctx, &AddActivityParams{Name: name, AthleteID: athleteID})
}

func (d *Repository) AddAthlete(ctx context.Context, name string, userID pgtype.UUID) (Athlete, error) {
	return d.q.AddAthlete(ctx, &AddAthleteParams{Name: name, UserID: userID})
}

func (d *Repository) AddMeasure(ctx context.Context, activityID pgtype.UUID, mmolPerLiter decimal.Decimal, heartRateBpm int32) (BloodLactateMeasure, error) {
	return d.q.AddMeasure(ctx, &AddMeasureParams{
		ActivityID:   activityID,
		MmolPerLiter: mmolPerLiter,
		HeartRateBpm: int32(heartRateBpm),
	})
}

func (d *Repository) DeleteAthlete(ctx context.Context, id pgtype.UUID) (Athlete, error) {
	return d.q.DeleteAthlete(ctx, id)
}
