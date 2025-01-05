package athletes

import (
	"context"
	"fmt"

	"github.com/catzkorn/trail-tools/internal/athletes/internal"
	"github.com/jackc/pgx/v5/pgtype"
)

type Athlete struct {
	*internal.Athlete
}

func (d *Repository) AddAthlete(ctx context.Context, name string, userID pgtype.UUID) (*Athlete, error) {
	athlete, err := d.q.AddAthlete(ctx, &internal.AddAthleteParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add athlete: %w", err)
	}
	return &Athlete{
		Athlete: athlete,
	}, nil
}

func (d *Repository) DeleteAthlete(ctx context.Context, id pgtype.UUID) (*Athlete, error) {
	athlete, err := d.q.DeleteAthlete(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete athlete: %w", err)
	}
	return &Athlete{
		Athlete: athlete,
	}, nil
}
