package services_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect/v2"
	"github.com/catzkorn/trail-tools/internal/activities"
	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
	stravav1 "github.com/catzkorn/trail-tools/internal/gen/strava/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
	stravasvc "github.com/catzkorn/trail-tools/internal/services/strava"
	"github.com/catzkorn/trail-tools/internal/strava"
)

// fakeStrava is a Strava API with a list of activities. The first request
// for streams hits the rate limit.
type fakeStrava struct {
	mu          sync.Mutex
	activities  []*strava.Activity
	photos      map[int64][]*strava.Photo
	rateLimited bool
	// Whether the athlete revoked access.
	revoked     bool
	refreshes   int
	streamCalls int
}

func newFakeStrava() *fakeStrava {
	f := &fakeStrava{photos: map[int64][]*strava.Photo{}}
	start := time.Date(2026, 9, 7, 8, 0, 0, 0, time.UTC) // A Monday.
	for i := range 3 {
		a := &strava.Activity{
			ID:                 int64(1000 + i),
			Name:               fmt.Sprintf("Run %d", i),
			SportType:          "TrailRun",
			StartDate:          start.AddDate(0, 0, i*4),
			StartDateLocal:     start.AddDate(0, 0, i*4).Format(time.RFC3339),
			Timezone:           "(GMT+00:00) Europe/London",
			Distance:           10000 + float64(i)*1000,
			MovingTime:         3600,
			ElapsedTime:        3700,
			TotalElevationGain: 500,
			AverageCadence:     85,
		}
		a.Map.SummaryPolyline = geo.EncodePolyline([]geo.Point{{Lat: 0, Lng: 0}, {Lat: 0.01, Lng: 0}})
		a.Raw, _ = json.Marshal(a)
		f.activities = append(f.activities, a)
	}
	// The third activity is a manual entry without streams.
	f.activities[2].Map.SummaryPolyline = ""
	// The first activity has two photos, one with a location.
	f.activities[0].TotalPhotoCount = 2
	f.photos[1000] = []*strava.Photo{
		{
			UniqueID: "p1", Caption: "Summit", URL: "https://strava.example/p1.jpg", Width: 2048, Height: 1536,
			Lat: 0.005, Lng: 0, HasLocation: true, CreatedAt: start.Add(30 * time.Minute), Raw: []byte("{}"),
		},
		{UniqueID: "p2", URL: "https://strava.example/p2.jpg", CreatedAt: start.Add(45 * time.Minute), Raw: []byte("{}")},
	}
	return f
}

func (f *fakeStrava) Refresh(_ context.Context, refreshToken string) (*strava.Token, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.revoked {
		return nil, strava.ErrUnauthorized
	}
	f.refreshes++
	return &strava.Token{AccessToken: "fresh", RefreshToken: refreshToken, Expiry: time.Now().Add(6 * time.Hour)}, nil
}

func (f *fakeStrava) ListActivities(_ context.Context, token string, after time.Time, page, perPage int) ([]*strava.Activity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if token != "fresh" {
		return nil, strava.ErrUnauthorized
	}
	if page > 1 {
		return nil, nil
	}
	var result []*strava.Activity
	for _, a := range f.activities {
		if a.StartDate.After(after) {
			result = append(result, a)
		}
	}
	return result, nil
}

func (f *fakeStrava) GetActivity(_ context.Context, _ string, id int64) (*strava.Activity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, a := range f.activities {
		if a.ID == id {
			detail := *a
			detail.Description = "Felt good"
			return &detail, nil
		}
	}
	return nil, strava.ErrNotFound
}

func (f *fakeStrava) GetStreams(_ context.Context, _ string, id int64) (*strava.Streams, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.streamCalls++
	if !f.rateLimited {
		f.rateLimited = true
		return nil, &strava.RateLimitError{ResetAt: time.Now()}
	}
	if id == 1002 {
		return nil, strava.ErrNotFound
	}
	streams := &strava.Streams{}
	for i := range 101 {
		streams.Time = append(streams.Time, int32(i*36))
		streams.Distance = append(streams.Distance, float64(i)*100)
		streams.LatLng = append(streams.LatLng, [2]float64{float64(i) * 0.0009, 0})
		streams.Altitude = append(streams.Altitude, float64(i)*5)
		streams.Cadence = append(streams.Cadence, 85)
	}
	return streams, nil
}

// connectStrava stores a Strava connection with an expired token for the
// user.
func (e *env) connectStrava() {
	e.t.Helper()
	if _, err := e.activities.UpsertStravaConnection(context.Background(), &activities.StravaConnection{
		UserID:       e.userID,
		AthleteID:    7,
		AthleteName:  "Test Runner",
		AccessToken:  "expired",
		RefreshToken: "refresh",
		Expiry:       time.Now().Add(-time.Hour),
	}); err != nil {
		e.t.Fatal(err)
	}
}

// waitForSync waits until the sync is idle.
func (e *env) waitForSync() *stravav1.StravaConnection {
	e.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := e.stravaAPI.GetStravaConnection(context.Background(), stravav1.GetStravaConnectionRequest_builder{}.Build())
		if err != nil {
			e.t.Fatal(err)
		}
		conn := resp.GetConnection()
		if !e.syncer.Running(e.userID) && conn.GetSyncState() != stravav1.SyncState_SYNC_STATE_RUNNING {
			return conn
		}
		time.Sleep(50 * time.Millisecond)
	}
	e.t.Fatal("sync did not finish")
	return nil
}

func TestStravaSyncAndActivities(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()

	_, err := e.stravaAPI.SyncStrava(ctx, stravav1.SyncStravaRequest_builder{}.Build())
	wantCode(t, err, connect.CodeFailedPrecondition)

	e.connectStrava()
	if _, err := e.stravaAPI.SyncStrava(ctx, stravav1.SyncStravaRequest_builder{}.Build()); err != nil {
		t.Fatalf("SyncStrava: %v", err)
	}
	conn := e.waitForSync()
	if conn.GetSyncState() != stravav1.SyncState_SYNC_STATE_IDLE || conn.GetActivityCount() != 3 || conn.GetPendingStreamCount() != 0 {
		t.Fatalf("unexpected connection after sync %v", conn)
	}
	if conn.GetAthleteName() != "Test Runner" || !conn.HasLastSyncTime() {
		t.Errorf("unexpected connection %v", conn)
	}
	// The expired token was refreshed, and the rate limited request retried.
	if e.strava.refreshes != 1 || e.strava.streamCalls != 4 {
		t.Errorf("got %d refreshes and %d stream calls, want 1 and 4", e.strava.refreshes, e.strava.streamCalls)
	}

	// Activities are listed most recent first, paginated.
	page1, err := e.activity.ListActivities(ctx, activitiesv1.ListActivitiesRequest_builder{PageSize: 2}.Build())
	if err != nil {
		t.Fatalf("ListActivities: %v", err)
	}
	if len(page1.GetActivities()) != 2 || page1.GetActivities()[0].GetName() != "Run 2" || page1.GetNextPageToken() == "" {
		t.Fatalf("unexpected first page %v", page1)
	}
	page2, err := e.activity.ListActivities(ctx, activitiesv1.ListActivitiesRequest_builder{
		PageSize:  2,
		PageToken: page1.GetNextPageToken(),
	}.Build())
	if err != nil {
		t.Fatalf("ListActivities: %v", err)
	}
	if len(page2.GetActivities()) != 1 || page2.GetNextPageToken() != "" {
		t.Fatalf("unexpected second page %v", page2)
	}
	run0 := page2.GetActivities()[0]
	if run0.GetDescription() != "Felt good" || !run0.GetHasStreams() || run0.GetAverageCadenceSpm() != 170 ||
		run0.GetStartDateLocal() != "2026-09-07" || run0.GetTimezone() != "Europe/London" {
		t.Errorf("unexpected activity %v", run0)
	}
	filtered, err := e.activity.ListActivities(ctx, activitiesv1.ListActivitiesRequest_builder{
		StartDate:  "2026-09-08",
		EndDate:    "2026-09-12",
		SportTypes: []string{"TrailRun"},
	}.Build())
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.GetActivities()) != 1 || filtered.GetActivities()[0].GetName() != "Run 1" {
		t.Errorf("unexpected filtered activities %v", filtered.GetActivities())
	}

	// Streams are downsampled.
	detail, err := e.activity.GetActivity(ctx, activitiesv1.GetActivityRequest_builder{
		Id:              run0.GetId(),
		IncludeStreams:  true,
		MaxStreamPoints: 11,
	}.Build())
	if err != nil {
		t.Fatalf("GetActivity: %v", err)
	}
	streams := detail.GetStreams()
	if len(streams.GetLat()) != 11 || streams.GetAltitudeMeters()[10] != 500 || streams.GetCadenceSpm()[0] != 170 {
		t.Errorf("unexpected streams %v", streams)
	}
	// Photos are synced with the details.
	photos := detail.GetPhotos()
	if run0.GetPhotoCount() != 2 || len(photos) != 2 || photos[0].GetId() != "p1" || photos[0].GetCaption() != "Summit" ||
		photos[0].GetLocation().GetLat() != 0.005 || photos[0].GetWidth() != 2048 || photos[1].HasLocation() ||
		!photos[1].GetTakenTime().AsTime().Equal(time.Date(2026, 9, 7, 8, 45, 0, 0, time.UTC)) {
		t.Errorf("unexpected photos %v", photos)
	}

	// Weekly summaries start on Monday and include empty weeks.
	summary, err := e.activity.SummarizeActivities(ctx, activitiesv1.SummarizeActivitiesRequest_builder{
		StartDate: "2026-09-09",
		EndDate:   "2026-09-27",
	}.Build())
	if err != nil {
		t.Fatalf("SummarizeActivities: %v", err)
	}
	weeks := summary.GetSummaries()
	if len(weeks) != 3 || weeks[0].GetStartDate() != "2026-09-07" || weeks[0].GetActivityCount() != 2 ||
		weeks[0].GetDistanceMeters() != 21000 || weeks[1].GetActivityCount() != 1 || weeks[2].GetActivityCount() != 0 {
		t.Errorf("unexpected weekly summaries %v", weeks)
	}
	if weeks[0].GetLongestDistanceMeters() != 11000 || weeks[0].GetElevationGainMeters() != 1000 {
		t.Errorf("unexpected first week %v", weeks[0])
	}

	// An activity can become a route.
	route, err := e.routes.CreateRoute(ctx, routesv1.CreateRouteRequest_builder{
		ActivityId: new(run0.GetId()),
	}.Build())
	if err != nil {
		t.Fatalf("CreateRoute from activity: %v", err)
	}
	if route.GetRoute().GetName() != "Run 0" || route.GetRoute().GetSource() != routesv1.RouteSource_ROUTE_SOURCE_ACTIVITY ||
		!approx(route.GetRoute().GetStats().GetDistanceMeters(), 10007, 5) || !approx(route.GetRoute().GetStats().GetElevationGainMeters(), 500, 1) {
		t.Errorf("unexpected route %v", route.GetRoute())
	}
	// A manual activity has no track.
	_, err = e.routes.CreateRoute(ctx, routesv1.CreateRouteRequest_builder{
		ActivityId: new(page1.GetActivities()[0].GetId()),
	}.Build())
	wantCode(t, err, connect.CodeFailedPrecondition)
}

func TestScheduledStravaSync(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()
	e.connectStrava()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	syncer := stravasvc.NewSyncer(ctx, log, e.activities, e.strava, false)

	// Background syncs are disabled.
	if syncer.Start(e.userID, false) {
		t.Fatal("Start should do nothing without background syncs")
	}
	// The first sync stops at the rate limit instead of waiting.
	if !syncer.SyncOnce(ctx, e.userID, false) {
		t.Fatal("SyncOnce didn't run")
	}
	conn, err := e.activities.GetStravaConnection(ctx, e.userID)
	if err != nil {
		t.Fatal(err)
	}
	if conn.SyncState != activities.SyncStateRateLimited {
		t.Fatalf("got sync state %q, want rate limited", conn.SyncState)
	}
	// The rate limit has reset, so the scheduled sync continues it.
	if err := syncer.SyncDue(ctx); err != nil {
		t.Fatalf("SyncDue: %v", err)
	}
	if conn, err = e.activities.GetStravaConnection(ctx, e.userID); err != nil {
		t.Fatal(err)
	}
	total, pending, err := e.activities.CountActivities(ctx, e.userID)
	if err != nil {
		t.Fatal(err)
	}
	if conn.SyncState != activities.SyncStateIdle || total != 3 || pending != 0 {
		t.Errorf("got state %q with %d activities and %d pending, want idle, 3 and 0", conn.SyncState, total, pending)
	}
	// Recently synced connections aren't due.
	calls := e.strava.streamCalls
	if err := syncer.SyncDue(ctx); err != nil || e.strava.streamCalls != calls {
		t.Errorf("an up to date connection was synced again: %v", err)
	}
}
