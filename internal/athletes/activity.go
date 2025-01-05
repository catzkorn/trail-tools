package athletes

import (
	"context"
	"fmt"

	"github.com/catzkorn/trail-tools/internal/athletes/internal"
	"github.com/jackc/pgx/v5/pgtype"
)

type Activity struct {
	*internal.Activity
}

func (d *Repository) AddActivity(ctx context.Context, name string, athleteID pgtype.UUID) (*Activity, error) {
	activity, err := d.q.AddActivity(ctx, &internal.AddActivityParams{
		Name:      name,
		AthleteID: athleteID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add activity: %w", err)
	}
	return &Activity{
		Activity: activity,
	}, nil
}
