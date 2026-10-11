// Package activities stores activities synced from Strava and the Strava
// connection used to sync them.
package activities

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/catzkorn/trail-tools/internal/activities/internal"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Repository stores activities and Strava connections.
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

// Activity is a recorded activity.
type Activity struct {
	ID                  pgtype.UUID
	StravaID            int64
	Name                string
	Description         string
	SportType           string
	StartTime           time.Time
	StartDateLocal      time.Time
	Timezone            string
	DistanceMeters      float64
	MovingTimeSeconds   int32
	ElapsedTimeSeconds  int32
	ElevationGainMeters float64
	AverageSpeedMps     float64
	MaxSpeedMps         float64
	AverageHeartrateBpm float64
	MaxHeartrateBpm     float64
	AverageCadenceSpm   float64
	SummaryPolyline     string
	DeviceName          string
	RelativeEffort      int32
	StreamsSynced       bool
	// The number of photos on Strava.
	PhotoCount int32
}

// ActivitySummary is an activity as listed by Strava, used when syncing.
type ActivitySummary struct {
	StravaID            int64
	Name                string
	SportType           string
	StartTime           time.Time
	StartDateLocal      time.Time
	Timezone            string
	DistanceMeters      float64
	MovingTimeSeconds   int32
	ElapsedTimeSeconds  int32
	ElevationGainMeters float64
	AverageSpeedMps     float64
	MaxSpeedMps         float64
	AverageHeartrateBpm float64
	MaxHeartrateBpm     float64
	AverageCadenceSpm   float64
	SummaryPolyline     string
	RelativeEffort      int32
	PhotoCount          int32
	// The raw Strava response.
	RawJSON []byte
}

// UpsertActivity creates or updates an activity from Strava.
func (r *Repository) UpsertActivity(ctx context.Context, userID pgtype.UUID, a *ActivitySummary) (pgtype.UUID, error) {
	params := &internal.UpsertActivityParams{
		UserID:              userID,
		StravaID:            a.StravaID,
		Name:                a.Name,
		SportType:           a.SportType,
		StartTime:           pgtype.Timestamptz{Time: a.StartTime, Valid: true},
		StartDateLocal:      store.DateToPG(a.StartDateLocal),
		Timezone:            a.Timezone,
		DistanceMeters:      a.DistanceMeters,
		MovingTimeSeconds:   a.MovingTimeSeconds,
		ElapsedTimeSeconds:  a.ElapsedTimeSeconds,
		ElevationGainMeters: a.ElevationGainMeters,
		AverageSpeedMps:     a.AverageSpeedMps,
		MaxSpeedMps:         a.MaxSpeedMps,
		AverageHeartrateBpm: a.AverageHeartrateBpm,
		MaxHeartrateBpm:     a.MaxHeartrateBpm,
		AverageCadenceSpm:   a.AverageCadenceSpm,
		SummaryPolyline:     a.SummaryPolyline,
		RelativeEffort:      a.RelativeEffort,
		SummaryJSON:         a.RawJSON,
		PhotoCount:          a.PhotoCount,
	}
	if a.SummaryPolyline != "" {
		if points, err := geo.DecodePolyline(a.SummaryPolyline); err == nil {
			if b, ok := geo.BoundsOf(points); ok {
				params.MinLat, params.MinLng = pgtype.Float8{Float64: b.MinLat, Valid: true}, pgtype.Float8{Float64: b.MinLng, Valid: true}
				params.MaxLat, params.MaxLng = pgtype.Float8{Float64: b.MaxLat, Valid: true}, pgtype.Float8{Float64: b.MaxLng, Valid: true}
			}
		}
	}
	id, err := r.q.UpsertActivity(ctx, params)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("failed to upsert activity: %w", err)
	}
	return id, nil
}

// LatestActivityStartTime returns the start time of the user's most recent
// activity, or the zero time if there are none.
func (r *Repository) LatestActivityStartTime(ctx context.Context, userID pgtype.UUID) (time.Time, error) {
	t, err := r.q.GetLatestActivityStartTime(ctx, userID)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest activity: %w", err)
	}
	return t.Time, nil
}

// PendingStreams is an activity whose streams have not been synced.
type PendingStreams struct {
	ID       pgtype.UUID
	StravaID int64
}

// ListActivitiesWithoutStreams lists activities whose streams have not been
// synced yet, most recent first.
func (r *Repository) ListActivitiesWithoutStreams(ctx context.Context, userID pgtype.UUID, limit int32) ([]PendingStreams, error) {
	rows, err := r.q.ListActivitiesWithoutStreams(ctx, &internal.ListActivitiesWithoutStreamsParams{UserID: userID, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("failed to list activities without streams: %w", err)
	}
	pending := make([]PendingStreams, len(rows))
	for i, row := range rows {
		pending[i] = PendingStreams{ID: row.ID, StravaID: row.StravaID}
	}
	return pending, nil
}

// CountActivities returns the number of activities and the number waiting
// for streams.
func (r *Repository) CountActivities(ctx context.Context, userID pgtype.UUID) (total, pendingStreams int64, err error) {
	row, err := r.q.CountActivities(ctx, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count activities: %w", err)
	}
	return row.Total, row.PendingStreams, nil
}

// SetActivityDetail stores details only available from Strava's detailed
// activity endpoint.
func (r *Repository) SetActivityDetail(ctx context.Context, activityID pgtype.UUID, description, deviceName string, rawJSON []byte) error {
	if err := r.q.SetActivityDetail(ctx, &internal.SetActivityDetailParams{
		ID:          activityID,
		Description: description,
		DeviceName:  deviceName,
		DetailJSON:  rawJSON,
	}); err != nil {
		return fmt.Errorf("failed to set activity detail: %w", err)
	}
	return nil
}

// SetActivityStreams stores the encoded streams of an activity and marks its
// streams as synced. streams may be nil if the activity has no streams.
func (r *Repository) SetActivityStreams(ctx context.Context, activityID pgtype.UUID, streams []byte, bounds *geo.Bounds) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(context.Background())
	q := r.q.WithTx(tx)
	params := &internal.MarkActivityStreamsSyncedParams{ID: activityID}
	if bounds != nil {
		params.MinLat, params.MinLng = pgtype.Float8{Float64: bounds.MinLat, Valid: true}, pgtype.Float8{Float64: bounds.MinLng, Valid: true}
		params.MaxLat, params.MaxLng = pgtype.Float8{Float64: bounds.MaxLat, Valid: true}, pgtype.Float8{Float64: bounds.MaxLng, Valid: true}
	}
	if err := q.MarkActivityStreamsSynced(ctx, params); err != nil {
		return fmt.Errorf("failed to mark streams synced: %w", err)
	}
	if streams != nil {
		if err := q.UpsertActivityStreams(ctx, &internal.UpsertActivityStreamsParams{ActivityID: activityID, Streams: streams}); err != nil {
			return fmt.Errorf("failed to store streams: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// ListActivitiesFilter filters listed activities.
type ListActivitiesFilter struct {
	// Inclusive local date range. Zero values are unbounded.
	StartDate, EndDate time.Time
	// Sport types to include. Empty includes all.
	SportTypes []string
	Limit      int32
	Offset     int32
}

// ListActivities lists the user's activities, most recent first.
func (r *Repository) ListActivities(ctx context.Context, userID pgtype.UUID, filter ListActivitiesFilter) ([]*Activity, error) {
	sportTypes := filter.SportTypes
	if sportTypes == nil {
		sportTypes = []string{}
	}
	rows, err := r.q.ListActivities(ctx, &internal.ListActivitiesParams{
		UserID:     userID,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
		StartDate:  store.DateToPG(filter.StartDate),
		EndDate:    store.DateToPG(filter.EndDate),
		SportTypes: sportTypes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list activities: %w", err)
	}
	result := make([]*Activity, len(rows))
	for i, row := range rows {
		result[i] = (*activityRow)(row).toActivity()
	}
	return result, nil
}

// GetActivity gets one of the user's activities.
func (r *Repository) GetActivity(ctx context.Context, userID, activityID pgtype.UUID) (*Activity, error) {
	row, err := r.q.GetActivity(ctx, &internal.GetActivityParams{ID: activityID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}
	return (*activityRow)(row).toActivity(), nil
}

// GetActivityStreams gets the encoded streams of one of the user's
// activities. It returns store.ErrNotFound if the activity has no streams.
func (r *Repository) GetActivityStreams(ctx context.Context, userID, activityID pgtype.UUID) ([]byte, error) {
	streams, err := r.q.GetActivityStreams(ctx, &internal.GetActivityStreamsParams{ActivityID: activityID, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get activity streams: %w", err)
	}
	return streams, nil
}

// ActivityTotals are the totals of a single activity, used for summaries.
type ActivityTotals struct {
	StartDateLocal      time.Time
	DistanceMeters      float64
	MovingTimeSeconds   int32
	ElevationGainMeters float64
}

// ListActivityTotals lists the totals of each activity in an inclusive date
// range, in date order.
func (r *Repository) ListActivityTotals(ctx context.Context, userID pgtype.UUID, startDate, endDate time.Time, sportTypes []string) ([]ActivityTotals, error) {
	if sportTypes == nil {
		sportTypes = []string{}
	}
	rows, err := r.q.ListActivityTotals(ctx, &internal.ListActivityTotalsParams{
		UserID:     userID,
		StartDate:  store.DateToPG(startDate),
		EndDate:    store.DateToPG(endDate),
		SportTypes: sportTypes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list activity totals: %w", err)
	}
	totals := make([]ActivityTotals, len(rows))
	for i, row := range rows {
		totals[i] = ActivityTotals{
			StartDateLocal:      store.DateFromPG(row.StartDateLocal),
			DistanceMeters:      row.DistanceMeters,
			MovingTimeSeconds:   row.MovingTimeSeconds,
			ElevationGainMeters: row.ElevationGainMeters,
		}
	}
	return totals, nil
}

// Track is the GPS track of an activity. Streams is nil if the streams have
// not been synced, in which case the summary polyline should be used.
type Track struct {
	ActivityID      pgtype.UUID
	SummaryPolyline string
	Streams         []byte
}

// ListTracksInBounds lists the tracks of activities whose bounding box
// intersects the bounds, optionally within an inclusive date range.
func (r *Repository) ListTracksInBounds(ctx context.Context, userID pgtype.UUID, bounds geo.Bounds, startDate, endDate time.Time) ([]Track, error) {
	rows, err := r.q.ListActivityTracksInBounds(ctx, &internal.ListActivityTracksInBoundsParams{
		UserID:    userID,
		North:     bounds.MaxLat,
		South:     bounds.MinLat,
		East:      bounds.MaxLng,
		West:      bounds.MinLng,
		StartDate: store.DateToPG(startDate),
		EndDate:   store.DateToPG(endDate),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tracks: %w", err)
	}
	tracks := make([]Track, len(rows))
	for i, row := range rows {
		tracks[i] = Track{ActivityID: row.ID, SummaryPolyline: row.SummaryPolyline, Streams: row.Streams}
	}
	return tracks, nil
}

// activityRow has the same fields as the rows returned by the activity
// queries, so they can be converted to it.
type activityRow struct {
	ID                  pgtype.UUID
	StravaID            int64
	Name                string
	Description         string
	SportType           string
	StartTime           pgtype.Timestamptz
	StartDateLocal      pgtype.Date
	Timezone            string
	DistanceMeters      float64
	MovingTimeSeconds   int32
	ElapsedTimeSeconds  int32
	ElevationGainMeters float64
	AverageSpeedMps     float64
	MaxSpeedMps         float64
	AverageHeartrateBpm float64
	MaxHeartrateBpm     float64
	AverageCadenceSpm   float64
	SummaryPolyline     string
	DeviceName          string
	RelativeEffort      int32
	StreamsSynced       bool
	PhotoCount          int32
}

func (row *activityRow) toActivity() *Activity {
	return &Activity{
		ID:                  row.ID,
		StravaID:            row.StravaID,
		Name:                row.Name,
		Description:         row.Description,
		SportType:           row.SportType,
		StartTime:           row.StartTime.Time,
		StartDateLocal:      store.DateFromPG(row.StartDateLocal),
		Timezone:            row.Timezone,
		DistanceMeters:      row.DistanceMeters,
		MovingTimeSeconds:   row.MovingTimeSeconds,
		ElapsedTimeSeconds:  row.ElapsedTimeSeconds,
		ElevationGainMeters: row.ElevationGainMeters,
		AverageSpeedMps:     row.AverageSpeedMps,
		MaxSpeedMps:         row.MaxSpeedMps,
		AverageHeartrateBpm: row.AverageHeartrateBpm,
		MaxHeartrateBpm:     row.MaxHeartrateBpm,
		AverageCadenceSpm:   row.AverageCadenceSpm,
		SummaryPolyline:     row.SummaryPolyline,
		DeviceName:          row.DeviceName,
		RelativeEffort:      row.RelativeEffort,
		StreamsSynced:       row.StreamsSynced,
		PhotoCount:          row.PhotoCount,
	}
}
