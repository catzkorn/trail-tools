// Package races stores races and race plans.
package races

import (
	"context"
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

// DeleteRace deletes a race and its race plans.
func (r *Repository) DeleteRace(ctx context.Context, userID, raceID pgtype.UUID) error {
	n, err := r.q.DeleteRace(ctx, &internal.DeleteRaceParams{ID: raceID, UserID: userID})
	if err != nil {
		return fmt.Errorf("failed to delete race: %w", err)
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
