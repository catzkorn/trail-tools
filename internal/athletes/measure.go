package athletes

import (
	"context"

	"github.com/catzkorn/trail-tools/internal/athletes/internal"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type BloodLactateMeasure struct {
	*internal.BloodLactateMeasure
}

func (d *Repository) AddMeasure(ctx context.Context, activityID pgtype.UUID, mmolPerLiter decimal.Decimal, heartRateBpm int32) (*BloodLactateMeasure, error) {
	measure, err := d.q.AddMeasure(ctx, &internal.AddMeasureParams{
		ActivityID:   activityID,
		MmolPerLiter: mmolPerLiter,
		HeartRateBpm: int32(heartRateBpm),
	})
	if err != nil {
		return nil, err
	}
	return &BloodLactateMeasure{
		BloodLactateMeasure: measure,
	}, nil
}
