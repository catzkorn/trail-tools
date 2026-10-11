package strava

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/catzkorn/trail-tools/internal/activities"
	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/strava"
	"github.com/jackc/pgx/v5/pgtype"
	"gitlab.com/greyxor/slogor"
	"google.golang.org/protobuf/proto"
)

// SyncRepository stores synced activities and the sync state.
type SyncRepository interface {
	GetStravaConnection(ctx context.Context, userID pgtype.UUID) (*activities.StravaConnection, error)
	ListStravaConnections(ctx context.Context) ([]*activities.StravaConnection, error)
	UpdateStravaTokens(ctx context.Context, userID pgtype.UUID, accessToken, refreshToken string, expiry time.Time) error
	UpdateStravaSyncState(ctx context.Context, userID pgtype.UUID, state activities.SyncState, message string, resumeTime, lastSyncTime time.Time) error
	UpsertActivity(ctx context.Context, userID pgtype.UUID, a *activities.ActivitySummary) (pgtype.UUID, error)
	LatestActivityStartTime(ctx context.Context, userID pgtype.UUID) (time.Time, error)
	ListActivitiesWithoutStreams(ctx context.Context, userID pgtype.UUID, limit int32) ([]activities.PendingStreams, error)
	CountActivities(ctx context.Context, userID pgtype.UUID) (total, pendingStreams int64, err error)
	SetActivityDetail(ctx context.Context, activityID pgtype.UUID, description, deviceName string, rawJSON []byte) error
	SetActivityStreams(ctx context.Context, activityID pgtype.UUID, streams []byte, bounds *geo.Bounds) error
}

// StravaClient is the Strava API.
type StravaClient interface {
	Refresh(ctx context.Context, refreshToken string) (*strava.Token, error)
	ListActivities(ctx context.Context, accessToken string, after time.Time, page, perPage int) ([]*strava.Activity, error)
	GetActivity(ctx context.Context, accessToken string, id int64) (*strava.Activity, error)
	GetStreams(ctx context.Context, accessToken string, id int64) (*strava.Streams, error)
}

const (
	// recentOverlap is how far before the latest synced activity an
	// incremental sync starts, to pick up recent edits.
	recentOverlap = 7 * 24 * time.Hour
	// autoSyncAge is the age of the last sync after which a background sync
	// starts automatically.
	autoSyncAge = 6 * time.Hour
	// autoSyncCheckInterval is how often to check for stale syncs.
	autoSyncCheckInterval = time.Hour
	// streamBatchSize is the number of activities fetched streams for between
	// progress updates.
	streamBatchSize = 25
)

// Syncer syncs activities from Strava in the background. At most one sync
// runs per user.
type Syncer struct {
	log    *slog.Logger
	repo   SyncRepository
	client StravaClient
	nowFn  func() time.Time

	// Whether syncs run in background goroutines. Otherwise, syncs only run
	// within SyncOnce and SyncDue calls.
	background bool

	mu      sync.Mutex
	ctx     context.Context
	running map[pgtype.UUID]bool
	wg      sync.WaitGroup
}

// NewSyncer creates a new Syncer. Syncs are stopped when ctx is cancelled.
// If background is false, Start does nothing and syncs must be run with
// SyncOnce and SyncDue, e.g. from a scheduled job.
func NewSyncer(ctx context.Context, log *slog.Logger, repo SyncRepository, client StravaClient, background bool) *Syncer {
	return &Syncer{
		log:        log,
		repo:       repo,
		client:     client,
		nowFn:      time.Now,
		background: background,
		ctx:        ctx,
		running:    map[pgtype.UUID]bool{},
	}
}

// Background returns whether syncs run in the background.
func (s *Syncer) Background() bool {
	return s.background
}

// Start starts syncing the user's activities in the background, unless a
// sync is already running or background syncs are disabled. It returns
// whether a new sync was started.
func (s *Syncer) Start(userID pgtype.UUID, full bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.background || s.running[userID] || s.ctx.Err() != nil {
		return false
	}
	s.running[userID] = true
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			s.mu.Lock()
			delete(s.running, userID)
			s.mu.Unlock()
		}()
		s.sync(s.ctx, userID, full, true)
	}()
	return true
}

// SyncOnce syncs the user's activities until done, the rate limit is
// reached or ctx is done, without waiting for rate limits to reset. It is
// used where background work isn't possible, such as on Cloud Run with
// request based billing. It returns false if a sync is already running.
func (s *Syncer) SyncOnce(ctx context.Context, userID pgtype.UUID, full bool) bool {
	s.mu.Lock()
	if s.running[userID] {
		s.mu.Unlock()
		return false
	}
	s.running[userID] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, userID)
		s.mu.Unlock()
	}()
	s.sync(ctx, userID, full, false)
	return true
}

// SyncDue syncs the connections that are due a sync, one at a time, until
// ctx is done. A sync is due if it was interrupted, the rate limit has reset
// or the last sync is old.
func (s *Syncer) SyncDue(ctx context.Context) error {
	conns, err := s.repo.ListStravaConnections(ctx)
	if err != nil {
		return err
	}
	for _, conn := range conns {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if s.due(conn) {
			s.SyncOnce(ctx, conn.UserID, false)
		}
	}
	return nil
}

// due returns whether a connection is due a sync.
func (s *Syncer) due(conn *activities.StravaConnection) bool {
	now := s.nowFn()
	switch conn.SyncState {
	case activities.SyncStateRunning:
		return true
	case activities.SyncStateRateLimited:
		return now.After(conn.ResumeTime)
	case activities.SyncStateFailed:
		return false
	default:
		return now.Sub(conn.LastSyncTime) > autoSyncAge
	}
}

// Running returns whether a sync is running for the user.
func (s *Syncer) Running(userID pgtype.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running[userID]
}

// Wait waits for all running syncs to stop.
func (s *Syncer) Wait() {
	s.wg.Wait()
}

// RunBackground resumes interrupted syncs and periodically syncs connections
// that haven't synced recently, until ctx is cancelled.
func (s *Syncer) RunBackground(ctx context.Context) {
	ticker := time.NewTicker(autoSyncCheckInterval)
	defer ticker.Stop()
	for {
		conns, err := s.repo.ListStravaConnections(ctx)
		if err != nil {
			s.log.ErrorContext(ctx, "Failed to list Strava connections", slogor.Err(err))
		}
		for _, conn := range conns {
			// Rate limited syncs resume waiting for the limit to reset.
			if conn.SyncState == activities.SyncStateRateLimited || s.due(conn) {
				s.Start(conn.UserID, false)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// errRateLimited is returned when a sync stops at the rate limit rather than
// waiting for it to reset.
var errRateLimited = errors.New("rate limited")

// sync runs a sync to completion, recording progress and errors in the
// connection's sync state. If wait is set, it waits for rate limits to
// reset, otherwise it stops.
func (s *Syncer) sync(ctx context.Context, userID pgtype.UUID, full, wait bool) {
	log := s.log.With(slog.String("user_id", store.UUIDToString(userID)))
	log.InfoContext(ctx, "Starting Strava sync", slog.Bool("full", full))
	err := s.runSync(ctx, log, userID, full, wait)
	if ctx.Err() != nil {
		// The server is shutting down or the time is up. Leave the state as
		// running so the sync resumes later.
		return
	}
	if errors.Is(err, errRateLimited) {
		// The rate limited state is already recorded.
		log.InfoContext(ctx, "Strava sync paused at the rate limit")
		return
	}
	// Use a fresh context so the final state is recorded.
	stateCtx := context.WithoutCancel(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Strava sync failed", slogor.Err(err))
		if err := s.repo.UpdateStravaSyncState(stateCtx, userID, activities.SyncStateFailed, err.Error(), time.Time{}, time.Time{}); err != nil {
			log.ErrorContext(ctx, "Failed to record sync failure", slogor.Err(err))
		}
		return
	}
	total, _, err := s.repo.CountActivities(stateCtx, userID)
	if err != nil {
		log.ErrorContext(ctx, "Failed to count activities", slogor.Err(err))
	}
	message := fmt.Sprintf("Synced %d activities", total)
	if err := s.repo.UpdateStravaSyncState(stateCtx, userID, activities.SyncStateIdle, message, time.Time{}, s.nowFn()); err != nil {
		log.ErrorContext(ctx, "Failed to record sync completion", slogor.Err(err))
	}
	log.InfoContext(ctx, "Finished Strava sync", slog.Int64("activities", total))
}

func (s *Syncer) runSync(ctx context.Context, log *slog.Logger, userID pgtype.UUID, full, wait bool) error {
	conn, err := s.repo.GetStravaConnection(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get Strava connection: %w", err)
	}
	if err := s.setState(ctx, userID, activities.SyncStateRunning, "Fetching activities", time.Time{}); err != nil {
		return err
	}

	// Fetch activity summaries.
	var after time.Time
	if !full {
		latest, err := s.repo.LatestActivityStartTime(ctx, userID)
		if err != nil {
			return err
		}
		if !latest.IsZero() {
			after = latest.Add(-recentOverlap)
		}
	}
	fetched := 0
	for page := 1; ; page++ {
		var list []*strava.Activity
		err := s.withToken(ctx, conn, wait, func(token string) error {
			var err error
			list, err = s.client.ListActivities(ctx, token, after, page, strava.MaxActivitiesPerPage)
			return err
		})
		if err != nil {
			return fmt.Errorf("failed to list activities: %w", err)
		}
		if len(list) == 0 {
			break
		}
		for _, a := range list {
			if _, err := s.repo.UpsertActivity(ctx, userID, summaryFromStrava(a)); err != nil {
				return err
			}
		}
		fetched += len(list)
		if err := s.setState(ctx, userID, activities.SyncStateRunning, fmt.Sprintf("Fetched %d activities", fetched), time.Time{}); err != nil {
			return err
		}
	}

	// Fetch details and streams, most recent first.
	for {
		pending, err := s.repo.ListActivitiesWithoutStreams(ctx, userID, streamBatchSize)
		if err != nil {
			return err
		}
		if len(pending) == 0 {
			return nil
		}
		_, remaining, err := s.repo.CountActivities(ctx, userID)
		if err != nil {
			return err
		}
		if err := s.setState(ctx, userID, activities.SyncStateRunning, fmt.Sprintf("Fetching activity details: %d remaining", remaining), time.Time{}); err != nil {
			return err
		}
		for _, p := range pending {
			if err := s.syncDetails(ctx, conn, p, wait); err != nil {
				return fmt.Errorf("failed to sync activity %d: %w", p.StravaID, err)
			}
		}
	}
}

// syncDetails fetches and stores the details and streams of an activity.
func (s *Syncer) syncDetails(ctx context.Context, conn *activities.StravaConnection, p activities.PendingStreams, wait bool) error {
	var detail *strava.Activity
	err := s.withToken(ctx, conn, wait, func(token string) error {
		var err error
		detail, err = s.client.GetActivity(ctx, token, p.StravaID)
		return err
	})
	switch {
	case errors.Is(err, strava.ErrNotFound):
		// The activity was deleted or made private on Strava.
	case err != nil:
		return err
	default:
		if err := s.repo.SetActivityDetail(ctx, p.ID, detail.Description, detail.DeviceName, detail.Raw); err != nil {
			return err
		}
	}

	var streams *strava.Streams
	err = s.withToken(ctx, conn, wait, func(token string) error {
		var err error
		streams, err = s.client.GetStreams(ctx, token, p.StravaID)
		return err
	})
	switch {
	case errors.Is(err, strava.ErrNotFound):
		// Manual activities have no streams.
		return s.repo.SetActivityStreams(ctx, p.ID, nil, nil)
	case err != nil:
		return err
	}
	footSport := detail != nil && detail.IsFootSport()
	encoded, bounds, err := encodeStreams(streams, footSport)
	if err != nil {
		return err
	}
	return s.repo.SetActivityStreams(ctx, p.ID, encoded, bounds)
}

// withToken calls fn with a valid access token, refreshing it if needed. If
// the rate limit is reached, it records the state, and either waits for the
// limit to reset and retries, or returns errRateLimited.
func (s *Syncer) withToken(ctx context.Context, conn *activities.StravaConnection, wait bool, fn func(token string) error) error {
	for {
		if s.nowFn().Add(5 * time.Minute).After(conn.Expiry) {
			token, err := s.client.Refresh(ctx, conn.RefreshToken)
			if err != nil {
				return err
			}
			if err := s.repo.UpdateStravaTokens(ctx, conn.UserID, token.AccessToken, token.RefreshToken, token.Expiry); err != nil {
				return err
			}
			conn.AccessToken, conn.RefreshToken, conn.Expiry = token.AccessToken, token.RefreshToken, token.Expiry
		}
		err := fn(conn.AccessToken)
		var rateErr *strava.RateLimitError
		if !errors.As(err, &rateErr) {
			return err
		}
		message := fmt.Sprintf("Paused until the Strava rate limit resets at %s", rateErr.ResetAt.UTC().Format("15:04 MST"))
		if err := s.setState(ctx, conn.UserID, activities.SyncStateRateLimited, message, rateErr.ResetAt); err != nil {
			return err
		}
		if !wait {
			return errRateLimited
		}
		timer := time.NewTimer(time.Until(rateErr.ResetAt) + 5*time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		if err := s.setState(ctx, conn.UserID, activities.SyncStateRunning, "Resuming", time.Time{}); err != nil {
			return err
		}
	}
}

func (s *Syncer) setState(ctx context.Context, userID pgtype.UUID, state activities.SyncState, message string, resumeTime time.Time) error {
	return s.repo.UpdateStravaSyncState(ctx, userID, state, message, resumeTime, time.Time{})
}

func summaryFromStrava(a *strava.Activity) *activities.ActivitySummary {
	cadence := a.AverageCadence
	if a.IsFootSport() {
		cadence *= 2
	}
	return &activities.ActivitySummary{
		StravaID:            a.ID,
		Name:                a.Name,
		SportType:           a.SportType,
		StartTime:           a.StartDate,
		StartDateLocal:      a.LocalDate(),
		Timezone:            a.TimezoneName(),
		DistanceMeters:      a.Distance,
		MovingTimeSeconds:   a.MovingTime,
		ElapsedTimeSeconds:  a.ElapsedTime,
		ElevationGainMeters: a.TotalElevationGain,
		AverageSpeedMps:     a.AverageSpeed,
		MaxSpeedMps:         a.MaxSpeed,
		AverageHeartrateBpm: a.AverageHeartrate,
		MaxHeartrateBpm:     a.MaxHeartrate,
		AverageCadenceSpm:   cadence,
		SummaryPolyline:     a.Map.SummaryPolyline,
		RelativeEffort:      int32(a.SufferScore),
		RawJSON:             a.Raw,
	}
}

// encodeStreams encodes streams as an activitiesv1.ActivityStreams message
// and returns the bounds of the GPS track, if any.
func encodeStreams(streams *strava.Streams, footSport bool) ([]byte, *geo.Bounds, error) {
	msg := activitiesv1.ActivityStreams_builder{
		TimeSeconds:    streams.Time,
		DistanceMeters: streams.Distance,
		AltitudeMeters: streams.Altitude,
		HeartrateBpm:   streams.Heartrate,
		VelocityMps:    streams.VelocitySmooth,
		GradePercent:   streams.GradeSmooth,
	}
	if len(streams.Cadence) > 0 {
		msg.CadenceSpm = make([]float64, len(streams.Cadence))
		for i, c := range streams.Cadence {
			if footSport {
				c *= 2
			}
			msg.CadenceSpm[i] = c
		}
	}
	var points []geo.Point
	if len(streams.LatLng) > 0 {
		msg.Lat = make([]float64, len(streams.LatLng))
		msg.Lng = make([]float64, len(streams.LatLng))
		points = make([]geo.Point, len(streams.LatLng))
		for i, ll := range streams.LatLng {
			msg.Lat[i], msg.Lng[i] = ll[0], ll[1]
			points[i] = geo.Point{Lat: ll[0], Lng: ll[1]}
		}
	}
	data, err := proto.Marshal(msg.Build())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to encode streams: %w", err)
	}
	bounds, ok := geo.BoundsOf(points)
	if !ok {
		return data, nil, nil
	}
	return data, &bounds, nil
}
