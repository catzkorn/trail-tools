// Package races stores races and race plans.
package races

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/catzkorn/trail-tools/internal/races/internal"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Repository stores races and race plans.
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

// Priority is how important a race is.
type Priority string

// Race priorities. The zero value means unset.
const (
	PriorityUnset Priority = ""
	PriorityA     Priority = "a"
	PriorityB     Priority = "b"
	PriorityC     Priority = "c"
)

// Race is a race.
type Race struct {
	ID                  pgtype.UUID
	UserID              pgtype.UUID
	Name                string
	Date                time.Time
	StartTime           string
	Location            string
	WebsiteURL          string
	DistanceMeters      float64
	ElevationGainMeters float64
	CutoffSeconds       int32
	Priority            Priority
	// Invalid if the race has no course.
	CourseRouteID pgtype.UUID
	Notes         string
	CreateTime    time.Time
	UpdateTime    time.Time
}

// CreateRace stores a new race.
func (r *Repository) CreateRace(ctx context.Context, race *Race) (*Race, error) {
	dbRace, err := r.q.CreateRace(ctx, &internal.CreateRaceParams{
		UserID:              race.UserID,
		Name:                race.Name,
		Date:                store.DateToPG(race.Date),
		StartTime:           race.StartTime,
		Location:            race.Location,
		WebsiteURL:          race.WebsiteURL,
		DistanceMeters:      race.DistanceMeters,
		ElevationGainMeters: race.ElevationGainMeters,
		CutoffSeconds:       race.CutoffSeconds,
		Priority:            priorityToDB(race.Priority),
		CourseRouteID:       race.CourseRouteID,
		Notes:               race.Notes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create race: %w", err)
	}
	return raceFromDB(dbRace), nil
}

// GetRace gets one of the user's races.
func (r *Repository) GetRace(ctx context.Context, userID, raceID pgtype.UUID) (*Race, error) {
	dbRace, err := r.q.GetRace(ctx, &internal.GetRaceParams{ID: raceID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get race: %w", err)
	}
	return raceFromDB(dbRace), nil
}

// ListRaces lists the user's races in date order.
func (r *Repository) ListRaces(ctx context.Context, userID pgtype.UUID) ([]*Race, error) {
	dbRaces, err := r.q.ListRaces(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list races: %w", err)
	}
	races := make([]*Race, len(dbRaces))
	for i, dbRace := range dbRaces {
		races[i] = raceFromDB(dbRace)
	}
	return races, nil
}

// UpdateRace replaces a race.
func (r *Repository) UpdateRace(ctx context.Context, race *Race) (*Race, error) {
	dbRace, err := r.q.UpdateRace(ctx, &internal.UpdateRaceParams{
		ID:                  race.ID,
		UserID:              race.UserID,
		Name:                race.Name,
		Date:                store.DateToPG(race.Date),
		StartTime:           race.StartTime,
		Location:            race.Location,
		WebsiteURL:          race.WebsiteURL,
		DistanceMeters:      race.DistanceMeters,
		ElevationGainMeters: race.ElevationGainMeters,
		CutoffSeconds:       race.CutoffSeconds,
		Priority:            priorityToDB(race.Priority),
		CourseRouteID:       race.CourseRouteID,
		Notes:               race.Notes,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update race: %w", err)
	}
	return raceFromDB(dbRace), nil
}

// DeleteRace deletes a race, its race plans and removes it from the goal
// races of the user's training plans.
func (r *Repository) DeleteRace(ctx context.Context, userID, raceID pgtype.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	q := r.q.WithTx(tx)
	n, err := q.DeleteRace(ctx, &internal.DeleteRaceParams{ID: raceID, UserID: userID})
	if err != nil {
		return fmt.Errorf("failed to delete race: %w", err)
	}
	if n == 0 {
		return store.ErrNotFound
	}
	if err := q.RemoveGoalRace(ctx, &internal.RemoveGoalRaceParams{UserID: userID, RaceID: raceID}); err != nil {
		return fmt.Errorf("failed to remove goal race from plans: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// RacePlan is a race plan. The plan's sections are stored as an opaque JSON
// document.
type RacePlan struct {
	ID       pgtype.UUID
	UserID   pgtype.UUID
	RaceID   pgtype.UUID
	Sections []byte
	// Empty if the plan is not shared.
	ShareToken string
	// The race day progress as an opaque JSON document, nil if there is none.
	RaceDay    []byte
	CreateTime time.Time
	UpdateTime time.Time
}

// CreateRacePlan stores a new race plan.
func (r *Repository) CreateRacePlan(ctx context.Context, userID, raceID pgtype.UUID, sections []byte) (*RacePlan, error) {
	plan, err := r.q.CreateRacePlan(ctx, &internal.CreateRacePlanParams{UserID: userID, RaceID: raceID, Sections: sections})
	if err != nil {
		return nil, fmt.Errorf("failed to create race plan: %w", err)
	}
	return planFromDB(plan), nil
}

// GetRacePlan gets one of the user's race plans.
func (r *Repository) GetRacePlan(ctx context.Context, userID, planID pgtype.UUID) (*RacePlan, error) {
	plan, err := r.q.GetRacePlan(ctx, &internal.GetRacePlanParams{ID: planID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get race plan: %w", err)
	}
	return planFromDB(plan), nil
}

// GetSharedRacePlan gets a shared race plan and its race by share token.
func (r *Repository) GetSharedRacePlan(ctx context.Context, token string) (*RacePlan, *Race, error) {
	if token == "" {
		return nil, nil, store.ErrNotFound
	}
	plan, err := r.q.GetRacePlanByShareToken(ctx, pgtype.Text{String: token, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, store.ErrNotFound
		}
		return nil, nil, fmt.Errorf("failed to get shared race plan: %w", err)
	}
	race, err := r.q.GetRaceByID(ctx, plan.RaceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get race: %w", err)
	}
	return planFromDB(plan), raceFromDB(race), nil
}

// ListRacePlans lists the user's race plans.
func (r *Repository) ListRacePlans(ctx context.Context, userID pgtype.UUID) ([]*RacePlan, error) {
	dbPlans, err := r.q.ListRacePlans(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list race plans: %w", err)
	}
	plans := make([]*RacePlan, len(dbPlans))
	for i, plan := range dbPlans {
		plans[i] = planFromDB(plan)
	}
	return plans, nil
}

// UpdateRacePlanSections replaces the sections of a race plan.
func (r *Repository) UpdateRacePlanSections(ctx context.Context, userID, planID pgtype.UUID, sections []byte) (*RacePlan, error) {
	plan, err := r.q.UpdateRacePlanSections(ctx, &internal.UpdateRacePlanSectionsParams{ID: planID, UserID: userID, Sections: sections})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to update race plan: %w", err)
	}
	return planFromDB(plan), nil
}

// ShareRacePlan creates a share token for a race plan, or returns the
// existing one.
func (r *Repository) ShareRacePlan(ctx context.Context, userID, planID pgtype.UUID) (*RacePlan, error) {
	plan, err := r.GetRacePlan(ctx, userID, planID)
	if err != nil {
		return nil, err
	}
	if plan.ShareToken != "" {
		return plan, nil
	}
	return r.setShareToken(ctx, userID, planID, pgtype.Text{String: rand.Text(), Valid: true})
}

// UnshareRacePlan removes a race plan's share token.
func (r *Repository) UnshareRacePlan(ctx context.Context, userID, planID pgtype.UUID) (*RacePlan, error) {
	return r.setShareToken(ctx, userID, planID, pgtype.Text{})
}

func (r *Repository) setShareToken(ctx context.Context, userID, planID pgtype.UUID, token pgtype.Text) (*RacePlan, error) {
	plan, err := r.q.SetRacePlanShareToken(ctx, &internal.SetRacePlanShareTokenParams{ID: planID, UserID: userID, ShareToken: token})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to set share token: %w", err)
	}
	return planFromDB(plan), nil
}

// SetRaceDay replaces the race day progress of a race plan. The caller must
// check the plan may be updated, e.g. that it belongs to the user or the
// share token matches.
func (r *Repository) SetRaceDay(ctx context.Context, planID pgtype.UUID, raceDay []byte) (*RacePlan, error) {
	plan, err := r.q.SetRacePlanRaceDay(ctx, &internal.SetRacePlanRaceDayParams{ID: planID, RaceDay: raceDay})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to set race day: %w", err)
	}
	return planFromDB(plan), nil
}

// DeleteRacePlan deletes one of the user's race plans.
func (r *Repository) DeleteRacePlan(ctx context.Context, userID, planID pgtype.UUID) error {
	n, err := r.q.DeleteRacePlan(ctx, &internal.DeleteRacePlanParams{ID: planID, UserID: userID})
	if err != nil {
		return fmt.Errorf("failed to delete race plan: %w", err)
	}
	if n == 0 {
		return store.ErrNotFound
	}
	return nil
}

func priorityToDB(p Priority) internal.NullRacePriority {
	if p == PriorityUnset {
		return internal.NullRacePriority{}
	}
	return internal.NullRacePriority{RacePriority: internal.RacePriority(p), Valid: true}
}

func raceFromDB(r *internal.Race) *Race {
	race := &Race{
		ID:                  r.ID,
		UserID:              r.UserID,
		Name:                r.Name,
		Date:                store.DateFromPG(r.Date),
		StartTime:           r.StartTime,
		Location:            r.Location,
		WebsiteURL:          r.WebsiteURL,
		DistanceMeters:      r.DistanceMeters,
		ElevationGainMeters: r.ElevationGainMeters,
		CutoffSeconds:       r.CutoffSeconds,
		CourseRouteID:       r.CourseRouteID,
		Notes:               r.Notes,
		CreateTime:          r.CreateTime.Time,
		UpdateTime:          r.UpdateTime.Time,
	}
	if r.Priority.Valid {
		race.Priority = Priority(r.Priority.RacePriority)
	}
	return race
}

func planFromDB(p *internal.RacePlan) *RacePlan {
	return &RacePlan{
		ID:         p.ID,
		UserID:     p.UserID,
		RaceID:     p.RaceID,
		Sections:   p.Sections,
		ShareToken: p.ShareToken.String,
		RaceDay:    p.RaceDay,
		CreateTime: p.CreateTime.Time,
		UpdateTime: p.UpdateTime.Time,
	}
}
