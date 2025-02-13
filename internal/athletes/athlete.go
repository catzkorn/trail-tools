package athletes

import (
	"context"
	"fmt"

	"github.com/catzkorn/trail-tools/internal/athletes/internal"
	"github.com/catzkorn/trail-tools/internal/store"
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

func (d *Repository) ListAthletesForUser(ctx context.Context, userID pgtype.UUID) ([]*Athlete, error) {
	athletes, err := d.q.ListAthletesForUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list athletes: %w", err)
	}
	var athleteResponses []*Athlete
	for _, athlete := range athletes {
		athleteResponses = append(athleteResponses, &Athlete{
			Athlete: athlete,
		})
	}
	return athleteResponses, nil
}

func (d *Repository) DeleteAthleteForUser(ctx context.Context, userID pgtype.UUID, athleteID pgtype.UUID) error {
	numRows, err := d.q.DeleteAthleteForUser(ctx, &internal.DeleteAthleteForUserParams{
		ID:     athleteID,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete athlete: %w", err)
	}
	if numRows == 0 {
		return store.ErrNotFound
	}
	return nil
}
