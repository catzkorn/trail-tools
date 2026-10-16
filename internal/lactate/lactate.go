// Package lactate stores the user's blood lactate tests, and finds their
// aerobic threshold in them.
package lactate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johanbrandhorst/trail-tools/internal/lactate/internal/gen"
	"github.com/johanbrandhorst/trail-tools/internal/store"
)

// AerobicThresholdMmolPerLiter is the blood lactate at the aerobic threshold,
// the definition Uphill Athlete uses.
const AerobicThresholdMmolPerLiter = 2.0

// Test is a blood lactate test.
type Test struct {
	ID     pgtype.UUID
	UserID pgtype.UUID
	Date   time.Time
	Notes  string
	// The resting blood lactate in mmol/L, 0 if not measured.
	RestingLactateMmolPerLiter float64
	// The stages, in the order they were run.
	Stages     []Stage
	CreateTime time.Time
	UpdateTime time.Time
}

// Stage is a stage of a test: a heart rate held for a few minutes, and the
// blood lactate measured at its end.
type Stage struct {
	HeartRateBPM        int32
	LactateMmolPerLiter float64
}

// LowestLactate returns the lowest reading of the test, at rest or of a
// stage, in mmol/L, or 0 if it has none.
func (t *Test) LowestLactate() float64 {
	lowest := t.RestingLactateMmolPerLiter
	for _, s := range t.Stages {
		if lowest == 0 || s.LactateMmolPerLiter < lowest {
			lowest = s.LactateMmolPerLiter
		}
	}
	return lowest
}

// HeartRateAt returns the heart rate at which the stages' lactate first rises
// to mmolPerLiter, interpolated between the stage below it and the stage
// reaching it, or 0 if no stage rises to it. Lactate often dips in the first
// stages after the warm-up, so a stage above it followed by one below doesn't
// count.
func (t *Test) HeartRateAt(mmolPerLiter float64) int32 {
	for i := 1; i < len(t.Stages); i++ {
		below, reached := t.Stages[i-1], t.Stages[i]
		if below.LactateMmolPerLiter >= mmolPerLiter || reached.LactateMmolPerLiter < mmolPerLiter {
			continue
		}
		f := (mmolPerLiter - below.LactateMmolPerLiter) / (reached.LactateMmolPerLiter - below.LactateMmolPerLiter)
		return int32(math.Round(float64(below.HeartRateBPM) + f*float64(reached.HeartRateBPM-below.HeartRateBPM)))
	}
	return 0
}

// Repository stores lactate tests.
type Repository struct {
	log *slog.Logger
	db  *store.DB
	q   *gen.Queries
}

// NewRepository creates a new Repository from the provided logger and
// database.
func NewRepository(log *slog.Logger, db *store.DB) (*Repository, error) {
	return &Repository{
		log: log,
		db:  db,
		q:   gen.New(db),
	}, nil
}

// CreateTest stores a new test.
func (r *Repository) CreateTest(ctx context.Context, test *Test) (*Test, error) {
	heartRates, lactates := stageColumns(test.Stages)
	dbTest, err := r.q.CreateLactateTest(ctx, &gen.CreateLactateTestParams{
		UserID:                     test.UserID,
		Date:                       store.DateToPG(test.Date),
		Notes:                      test.Notes,
		RestingLactateMmolPerLiter: restingToDB(test.RestingLactateMmolPerLiter),
		StageHeartRatesBpm:         heartRates,
		StageLactatesMmolPerLiter:  lactates,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create lactate test: %w", err)
	}
	return testFromDB(dbTest), nil
}

// GetTest gets one of the user's tests.
func (r *Repository) GetTest(ctx context.Context, userID, testID pgtype.UUID) (*Test, error) {
	dbTest, err := r.q.GetLactateTest(ctx, &gen.GetLactateTestParams{ID: testID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get lactate test: %w", err)
	}
	return testFromDB(dbTest), nil
}

// ListTests lists the user's tests, the latest first.
func (r *Repository) ListTests(ctx context.Context, userID pgtype.UUID) ([]*Test, error) {
	dbTests, err := r.q.ListLactateTests(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list lactate tests: %w", err)
	}
	tests := make([]*Test, len(dbTests))
	for i, dbTest := range dbTests {
		tests[i] = testFromDB(dbTest)
	}
	return tests, nil
}

// UpdateTest replaces a test.
func (r *Repository) UpdateTest(ctx context.Context, test *Test) (*Test, error) {
	heartRates, lactates := stageColumns(test.Stages)
	dbTest, err := r.q.UpdateLactateTest(ctx, &gen.UpdateLactateTestParams{
		ID:                         test.ID,
		UserID:                     test.UserID,
		Date:                       store.DateToPG(test.Date),
		Notes:                      test.Notes,
		RestingLactateMmolPerLiter: restingToDB(test.RestingLactateMmolPerLiter),
		StageHeartRatesBpm:         heartRates,
		StageLactatesMmolPerLiter:  lactates,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update lactate test: %w", err)
	}
	return testFromDB(dbTest), nil
}

// DeleteTest deletes one of the user's tests.
func (r *Repository) DeleteTest(ctx context.Context, userID, testID pgtype.UUID) error {
	n, err := r.q.DeleteLactateTest(ctx, &gen.DeleteLactateTestParams{ID: testID, UserID: userID})
	if err != nil {
		return fmt.Errorf("failed to delete lactate test: %w", err)
	}
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

// stageColumns returns the stages' heart rates and lactates, the columns
// they're stored in.
func stageColumns(stages []Stage) (heartRates []int32, lactates []float64) {
	heartRates, lactates = make([]int32, len(stages)), make([]float64, len(stages))
	for i, s := range stages {
		heartRates[i], lactates[i] = s.HeartRateBPM, s.LactateMmolPerLiter
	}
	return heartRates, lactates
}

func restingToDB(mmolPerLiter float64) pgtype.Float8 {
	return pgtype.Float8{Float64: mmolPerLiter, Valid: mmolPerLiter > 0}
}

func testFromDB(t *gen.LactateTest) *Test {
	stages := make([]Stage, len(t.StageHeartRatesBpm))
	for i := range stages {
		stages[i] = Stage{HeartRateBPM: t.StageHeartRatesBpm[i], LactateMmolPerLiter: t.StageLactatesMmolPerLiter[i]}
	}
	return &Test{
		ID:                         t.ID,
		UserID:                     t.UserID,
		Date:                       store.DateFromPG(t.Date),
		Notes:                      t.Notes,
		RestingLactateMmolPerLiter: t.RestingLactateMmolPerLiter.Float64,
		Stages:                     stages,
		CreateTime:                 t.CreateTime.Time,
		UpdateTime:                 t.UpdateTime.Time,
	}
}
