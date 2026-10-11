// Package plans stores training plans, their month and week periods, and
// workouts.
package plans

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/plans/internal"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Repository stores training plans.
type Repository struct {
	log *slog.Logger
	db  *store.DB
	q   *internal.Queries
}

// NewRepository creates a new Repository from the provided logger and database.
func NewRepository(log *slog.Logger, db *store.DB) (*Repository, error) {
	return &Repository{
		log: log,
		db:  db,
		q:   internal.New(db),
	}, nil
}

// TrainingPlan is a training plan.
type TrainingPlan struct {
	ID          pgtype.UUID
	UserID      pgtype.UUID
	Name        string
	Description string
	StartDate   time.Time
	EndDate     time.Time
	GoalRaceIDs []pgtype.UUID
	CreateTime  time.Time
	UpdateTime  time.Time
}

// Granularity is the length of a plan period.
type Granularity string

// Period granularities.
const (
	GranularityMonth Granularity = "month"
	GranularityWeek  Granularity = "week"
)

// Period is a month or week in a training plan.
type Period struct {
	ID                        pgtype.UUID
	PlanID                    pgtype.UUID
	Granularity               Granularity
	StartDate                 time.Time
	Title                     string
	Notes                     string
	TargetDistanceMeters      float64
	TargetElevationGainMeters float64
	TargetDurationSeconds     int32
}

// WorkoutType is the type of a workout.
type WorkoutType string

// Workout types.
const (
	WorkoutTypeEasy          WorkoutType = "easy"
	WorkoutTypeLongRun       WorkoutType = "long_run"
	WorkoutTypeTempo         WorkoutType = "tempo"
	WorkoutTypeIntervals     WorkoutType = "intervals"
	WorkoutTypeHillRepeats   WorkoutType = "hill_repeats"
	WorkoutTypeVert          WorkoutType = "vert"
	WorkoutTypeRecovery      WorkoutType = "recovery"
	WorkoutTypeRest          WorkoutType = "rest"
	WorkoutTypeCrossTraining WorkoutType = "cross_training"
	WorkoutTypeStrength      WorkoutType = "strength"
	WorkoutTypeRace          WorkoutType = "race"
	WorkoutTypeOther         WorkoutType = "other"
)

// Workout is a workout planned for a day.
type Workout struct {
	ID                        pgtype.UUID
	PlanID                    pgtype.UUID
	Date                      time.Time
	Type                      WorkoutType
	Title                     string
	Description               string
	TargetDistanceMeters      float64
	TargetElevationGainMeters float64
	TargetDurationSeconds     int32
	// Unknown if there is no target terrain.
	TargetTerrain geo.Terrain
	// Invalid if the workout has no route.
	RouteID pgtype.UUID
	// Invalid if the workout has not been completed.
	CompletedActivityID pgtype.UUID
}

// CreateTrainingPlan stores a new training plan.
func (r *Repository) CreateTrainingPlan(ctx context.Context, plan *TrainingPlan) (*TrainingPlan, error) {
	dbPlan, err := r.q.CreateTrainingPlan(ctx, &internal.CreateTrainingPlanParams{
		UserID:      plan.UserID,
		Name:        plan.Name,
		Description: plan.Description,
		StartDate:   store.DateToPG(plan.StartDate),
		EndDate:     store.DateToPG(plan.EndDate),
		GoalRaceIds: nonNil(plan.GoalRaceIDs),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create training plan: %w", err)
	}
	return planFromDB(dbPlan), nil
}

// GetTrainingPlan gets one of the user's training plans.
func (r *Repository) GetTrainingPlan(ctx context.Context, userID, planID pgtype.UUID) (*TrainingPlan, error) {
	dbPlan, err := r.q.GetTrainingPlan(ctx, &internal.GetTrainingPlanParams{ID: planID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get training plan: %w", err)
	}
	return planFromDB(dbPlan), nil
}

// ListTrainingPlans lists the user's training plans, most recent start date
// first.
func (r *Repository) ListTrainingPlans(ctx context.Context, userID pgtype.UUID) ([]*TrainingPlan, error) {
	dbPlans, err := r.q.ListTrainingPlans(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list training plans: %w", err)
	}
	result := make([]*TrainingPlan, len(dbPlans))
	for i, p := range dbPlans {
		result[i] = planFromDB(p)
	}
	return result, nil
}

// UpdateTrainingPlan replaces a training plan's details.
func (r *Repository) UpdateTrainingPlan(ctx context.Context, plan *TrainingPlan) (*TrainingPlan, error) {
	dbPlan, err := r.q.UpdateTrainingPlan(ctx, &internal.UpdateTrainingPlanParams{
		ID:          plan.ID,
		UserID:      plan.UserID,
		Name:        plan.Name,
		Description: plan.Description,
		StartDate:   store.DateToPG(plan.StartDate),
		EndDate:     store.DateToPG(plan.EndDate),
		GoalRaceIds: nonNil(plan.GoalRaceIDs),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update training plan: %w", err)
	}
	return planFromDB(dbPlan), nil
}

// DeleteTrainingPlan deletes a training plan with its periods and workouts.
func (r *Repository) DeleteTrainingPlan(ctx context.Context, userID, planID pgtype.UUID) error {
	n, err := r.q.DeleteTrainingPlan(ctx, &internal.DeleteTrainingPlanParams{ID: planID, UserID: userID})
	if err != nil {
		return fmt.Errorf("failed to delete training plan: %w", err)
	}
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

// SetPeriods creates or replaces periods of a plan, identified by their
// granularity and start date. The caller must check the plan belongs to the
// user.
func (r *Repository) SetPeriods(ctx context.Context, planID pgtype.UUID, periods []*Period) ([]*Period, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	q := r.q.WithTx(tx)
	result := make([]*Period, len(periods))
	for i, p := range periods {
		dbPeriod, err := q.UpsertPlanPeriod(ctx, &internal.UpsertPlanPeriodParams{
			PlanID:                    planID,
			Granularity:               internal.PeriodGranularity(p.Granularity),
			StartDate:                 store.DateToPG(p.StartDate),
			Title:                     p.Title,
			Notes:                     p.Notes,
			TargetDistanceMeters:      p.TargetDistanceMeters,
			TargetElevationGainMeters: p.TargetElevationGainMeters,
			TargetDurationSeconds:     p.TargetDurationSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to set period: %w", err)
		}
		result[i] = periodFromDB(dbPeriod)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return result, nil
}

// ListPeriods lists a plan's periods in date order. The caller must check the
// plan belongs to the user.
func (r *Repository) ListPeriods(ctx context.Context, planID pgtype.UUID) ([]*Period, error) {
	dbPeriods, err := r.q.ListPlanPeriods(ctx, planID)
	if err != nil {
		return nil, fmt.Errorf("failed to list periods: %w", err)
	}
	result := make([]*Period, len(dbPeriods))
	for i, p := range dbPeriods {
		result[i] = periodFromDB(p)
	}
	return result, nil
}

// DeletePeriods deletes periods of a plan. The caller must check the plan
// belongs to the user.
func (r *Repository) DeletePeriods(ctx context.Context, planID pgtype.UUID, ids []pgtype.UUID) error {
	if _, err := r.q.DeletePlanPeriods(ctx, &internal.DeletePlanPeriodsParams{PlanID: planID, Ids: ids}); err != nil {
		return fmt.Errorf("failed to delete periods: %w", err)
	}
	return nil
}

// CreateWorkouts creates workouts in a plan. The caller must check the plan
// belongs to the user.
func (r *Repository) CreateWorkouts(ctx context.Context, planID pgtype.UUID, workouts []*Workout) ([]*Workout, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	q := r.q.WithTx(tx)
	result := make([]*Workout, len(workouts))
	for i, w := range workouts {
		dbWorkout, err := q.CreateWorkout(ctx, &internal.CreateWorkoutParams{
			PlanID:                    planID,
			Date:                      store.DateToPG(w.Date),
			Type:                      internal.WorkoutType(w.Type),
			Title:                     w.Title,
			Description:               w.Description,
			TargetDistanceMeters:      w.TargetDistanceMeters,
			TargetElevationGainMeters: w.TargetElevationGainMeters,
			TargetDurationSeconds:     w.TargetDurationSeconds,
			TargetTerrain:             terrainToDB(w.TargetTerrain),
			RouteID:                   w.RouteID,
			CompletedActivityID:       w.CompletedActivityID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create workout: %w", err)
		}
		result[i] = workoutFromDB(dbWorkout)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	return result, nil
}

// ListWorkouts lists a plan's workouts in an inclusive date range, in date
// order. Zero dates are unbounded. The caller must check the plan belongs to
// the user.
func (r *Repository) ListWorkouts(ctx context.Context, planID pgtype.UUID, startDate, endDate time.Time) ([]*Workout, error) {
	dbWorkouts, err := r.q.ListWorkouts(ctx, &internal.ListWorkoutsParams{
		PlanID:    planID,
		StartDate: store.DateToPG(startDate),
		EndDate:   store.DateToPG(endDate),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list workouts: %w", err)
	}
	result := make([]*Workout, len(dbWorkouts))
	for i, w := range dbWorkouts {
		result[i] = workoutFromDB(w)
	}
	return result, nil
}

// GetWorkout gets one of the user's workouts.
func (r *Repository) GetWorkout(ctx context.Context, userID, workoutID pgtype.UUID) (*Workout, error) {
	w, err := r.q.GetWorkout(ctx, &internal.GetWorkoutParams{ID: workoutID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get workout: %w", err)
	}
	return workoutFromDB(w), nil
}

// UpdateWorkout replaces a workout. The caller must check the workout belongs
// to the user.
func (r *Repository) UpdateWorkout(ctx context.Context, w *Workout) (*Workout, error) {
	dbWorkout, err := r.q.UpdateWorkout(ctx, &internal.UpdateWorkoutParams{
		ID:                        w.ID,
		Date:                      store.DateToPG(w.Date),
		Type:                      internal.WorkoutType(w.Type),
		Title:                     w.Title,
		Description:               w.Description,
		TargetDistanceMeters:      w.TargetDistanceMeters,
		TargetElevationGainMeters: w.TargetElevationGainMeters,
		TargetDurationSeconds:     w.TargetDurationSeconds,
		TargetTerrain:             terrainToDB(w.TargetTerrain),
		RouteID:                   w.RouteID,
		CompletedActivityID:       w.CompletedActivityID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update workout: %w", err)
	}
	return workoutFromDB(dbWorkout), nil
}

// DeleteWorkouts deletes workouts of a plan. The caller must check the plan
// belongs to the user.
func (r *Repository) DeleteWorkouts(ctx context.Context, planID pgtype.UUID, ids []pgtype.UUID) error {
	if _, err := r.q.DeleteWorkouts(ctx, &internal.DeleteWorkoutsParams{PlanID: planID, Ids: ids}); err != nil {
		return fmt.Errorf("failed to delete workouts: %w", err)
	}
	return nil
}

// RouteExists checks whether a route exists and belongs to the user.
func (r *Repository) RouteExists(ctx context.Context, userID, routeID pgtype.UUID) (bool, error) {
	exists, err := r.q.RouteExists(ctx, &internal.RouteExistsParams{ID: routeID, UserID: userID})
	if err != nil {
		return false, fmt.Errorf("failed to check route: %w", err)
	}
	return exists, nil
}

// ActivityExists checks whether an activity exists and belongs to the user.
func (r *Repository) ActivityExists(ctx context.Context, userID, activityID pgtype.UUID) (bool, error) {
	exists, err := r.q.ActivityExists(ctx, &internal.ActivityExistsParams{ID: activityID, UserID: userID})
	if err != nil {
		return false, fmt.Errorf("failed to check activity: %w", err)
	}
	return exists, nil
}

// RacesExist checks whether all the races exist and belong to the user.
func (r *Repository) RacesExist(ctx context.Context, userID pgtype.UUID, raceIDs []pgtype.UUID) (bool, error) {
	if len(raceIDs) == 0 {
		return true, nil
	}
	n, err := r.q.CountRaces(ctx, &internal.CountRacesParams{UserID: userID, Ids: raceIDs})
	if err != nil {
		return false, fmt.Errorf("failed to check races: %w", err)
	}
	unique := map[pgtype.UUID]bool{}
	for _, id := range raceIDs {
		unique[id] = true
	}
	return int(n) == len(unique), nil
}

func nonNil(ids []pgtype.UUID) []pgtype.UUID {
	if ids == nil {
		return []pgtype.UUID{}
	}
	return ids
}

func terrainToDB(t geo.Terrain) internal.NullTerrain {
	if t == geo.TerrainUnknown {
		return internal.NullTerrain{}
	}
	return internal.NullTerrain{Terrain: internal.Terrain(t), Valid: true}
}

func planFromDB(p *internal.TrainingPlan) *TrainingPlan {
	return &TrainingPlan{
		ID:          p.ID,
		UserID:      p.UserID,
		Name:        p.Name,
		Description: p.Description,
		StartDate:   store.DateFromPG(p.StartDate),
		EndDate:     store.DateFromPG(p.EndDate),
		GoalRaceIDs: p.GoalRaceIds,
		CreateTime:  p.CreateTime.Time,
		UpdateTime:  p.UpdateTime.Time,
	}
}

func periodFromDB(p *internal.PlanPeriod) *Period {
	return &Period{
		ID:                        p.ID,
		PlanID:                    p.PlanID,
		Granularity:               Granularity(p.Granularity),
		StartDate:                 store.DateFromPG(p.StartDate),
		Title:                     p.Title,
		Notes:                     p.Notes,
		TargetDistanceMeters:      p.TargetDistanceMeters,
		TargetElevationGainMeters: p.TargetElevationGainMeters,
		TargetDurationSeconds:     p.TargetDurationSeconds,
	}
}

func workoutFromDB(w *internal.Workout) *Workout {
	workout := &Workout{
		ID:                        w.ID,
		PlanID:                    w.PlanID,
		Date:                      store.DateFromPG(w.Date),
		Type:                      WorkoutType(w.Type),
		Title:                     w.Title,
		Description:               w.Description,
		TargetDistanceMeters:      w.TargetDistanceMeters,
		TargetElevationGainMeters: w.TargetElevationGainMeters,
		TargetDurationSeconds:     w.TargetDurationSeconds,
		RouteID:                   w.RouteID,
		CompletedActivityID:       w.CompletedActivityID,
	}
	if w.TargetTerrain.Valid {
		workout.TargetTerrain = geo.Terrain(w.TargetTerrain.Terrain)
	}
	return workout
}
