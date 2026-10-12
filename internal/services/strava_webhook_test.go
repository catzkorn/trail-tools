package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/catzkorn/trail-tools/internal/activities"
	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	stravasvc "github.com/catzkorn/trail-tools/internal/services/strava"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/strava"
)

func (f *fakeStrava) GetActivityPhotos(_ context.Context, _ string, id int64) ([]*strava.Photo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.photos[id], nil
}

// upsertActivity adds or replaces an activity on Strava.
func (f *fakeStrava) upsertActivity(a *strava.Activity) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a.Raw, _ = json.Marshal(a)
	for i, existing := range f.activities {
		if existing.ID == a.ID {
			f.activities[i] = a
			return
		}
	}
	f.activities = append(f.activities, a)
}

// deleteActivity deletes an activity on Strava.
func (f *fakeStrava) deleteActivity(id int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, a := range f.activities {
		if a.ID == id {
			f.activities = append(f.activities[:i], f.activities[i+1:]...)
			return
		}
	}
}

// fakeSubscriptions is Strava's push subscriptions API.
type fakeSubscriptions struct {
	subs    []*strava.PushSubscription
	deleted []int64
	// Called while creating a subscription, like Strava validating the
	// callback.
	validate func(callbackURL, verifyToken string) error
}

func (f *fakeSubscriptions) ListPushSubscriptions(context.Context) ([]*strava.PushSubscription, error) {
	return f.subs, nil
}

func (f *fakeSubscriptions) CreatePushSubscription(_ context.Context, callbackURL, verifyToken string) (*strava.PushSubscription, error) {
	if err := f.validate(callbackURL, verifyToken); err != nil {
		return nil, err
	}
	sub := &strava.PushSubscription{ID: 100 + int64(len(f.deleted)), CallbackURL: callbackURL}
	f.subs = append(f.subs, sub)
	return sub, nil
}

func (f *fakeSubscriptions) DeletePushSubscription(_ context.Context, id int64) error {
	f.deleted = append(f.deleted, id)
	f.subs = nil
	return nil
}

func TestStravaWebhook(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	e.connectStrava()
	// Skip the fake's rate limit, which is tested with the full sync.
	e.strava.rateLimited = true
	subs := &fakeSubscriptions{}
	webhook := stravasvc.NewWebhook(log, e.activities, e.syncer, subs, "https://trail.example/strava/webhook", "verify-me")

	// Strava validates the callback with the verify token.
	validate := func(callbackURL, verifyToken string) error {
		q := url.Values{"hub.mode": {"subscribe"}, "hub.challenge": {"abc123"}, "hub.verify_token": {verifyToken}}
		rec := httptest.NewRecorder()
		webhook.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, callbackURL+"?"+q.Encode(), nil))
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"hub.challenge":"abc123"}` {
			return fmt.Errorf("callback not verifiable: %d %s", rec.Code, rec.Body)
		}
		return nil
	}
	if err := validate("https://trail.example/strava/webhook", "wrong"); err == nil {
		t.Error("validation with the wrong verify token should fail")
	}
	// A subscription to another URL is replaced, and an existing one kept.
	subs.validate = validate
	subs.subs = []*strava.PushSubscription{{ID: 1, CallbackURL: "https://old.example/strava/webhook"}}
	if err := webhook.Subscribe(ctx); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if len(subs.deleted) != 1 || len(subs.subs) != 1 || subs.subs[0].CallbackURL != "https://trail.example/strava/webhook" {
		t.Errorf("unexpected subscriptions %v, deleted %v", subs.subs, subs.deleted)
	}
	if err := webhook.Subscribe(ctx); err != nil || len(subs.subs) != 1 || len(subs.deleted) != 1 {
		t.Errorf("Subscribe with an existing subscription: %v, %v", err, subs.subs)
	}
	local := stravasvc.NewWebhook(log, e.activities, e.syncer, subs, "http://localhost:8080/strava/webhook", "verify-me")
	if err := local.Subscribe(ctx); err == nil {
		t.Error("Subscribe should fail without a public URL")
	}

	send := func(ev string) {
		t.Helper()
		rec := httptest.NewRecorder()
		webhook.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/strava/webhook", strings.NewReader(ev)))
		if rec.Code != http.StatusOK {
			t.Fatalf("event %s: got status %d", ev, rec.Code)
		}
	}
	activityEvent := func(aspect string, id, owner int64) string {
		return fmt.Sprintf(`{"aspect_type": %q, "event_time": 1757232000, "object_id": %d, "object_type": "activity",
			"owner_id": %d, "subscription_id": 100, "updates": {}}`, aspect, id, owner)
	}
	getActivity := func(stravaID int64) (*activitiesv1.GetActivityResponse, bool) {
		t.Helper()
		list, err := e.activities.ListActivities(ctx, e.userID, activities.ListActivitiesFilter{Limit: 100})
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range list {
			if a.StravaID == stravaID {
				resp, err := e.activity.GetActivity(ctx, activitiesv1.GetActivityRequest_builder{Id: store.UUIDToString(a.ID), IncludeStreams: true}.Build())
				if err != nil {
					t.Fatal(err)
				}
				return resp, true
			}
		}
		return nil, false
	}

	// A new activity is synced with its details, streams and photos.
	start := time.Date(2026, 10, 3, 7, 0, 0, 0, time.UTC)
	newActivity := &strava.Activity{
		ID:              2000,
		Name:            "Morning Run",
		SportType:       "TrailRun",
		StartDate:       start,
		StartDateLocal:  start.Format(time.RFC3339),
		Timezone:        "(GMT+00:00) Europe/London",
		Distance:        10000,
		MovingTime:      3600,
		ElapsedTime:     3600,
		TotalPhotoCount: 1,
	}
	e.strava.upsertActivity(newActivity)
	e.strava.mu.Lock()
	e.strava.photos[2000] = []*strava.Photo{{UniqueID: "q1", URL: "https://strava.example/q1.jpg", Raw: []byte("{}")}}
	e.strava.mu.Unlock()
	send(activityEvent("create", 2000, 7))
	got, ok := getActivity(2000)
	if !ok {
		t.Fatal("the created activity wasn't synced")
	}
	if got.GetActivity().GetName() != "Morning Run" || got.GetActivity().GetDescription() != "Felt good" ||
		!got.GetActivity().GetHasStreams() || len(got.GetStreams().GetLat()) == 0 || len(got.GetPhotos()) != 1 {
		t.Errorf("unexpected synced activity %v", got)
	}

	// Updates sync the activity again, and Strava sending the event again
	// is harmless.
	renamed := *newActivity
	renamed.Name = "Foggy Morning Run"
	renamed.TotalPhotoCount = 0
	e.strava.upsertActivity(&renamed)
	send(activityEvent("update", 2000, 7))
	send(activityEvent("update", 2000, 7))
	if got, _ := getActivity(2000); got.GetActivity().GetName() != "Foggy Morning Run" || len(got.GetPhotos()) != 0 || !got.GetActivity().GetHasStreams() {
		t.Errorf("unexpected updated activity %v", got)
	}

	// Events of other athletes are ignored.
	send(activityEvent("update", 1000, 8))
	// Events aren't trusted: an activity is only deleted once Strava no
	// longer has it.
	send(activityEvent("delete", 2000, 7))
	if _, ok := getActivity(2000); !ok {
		t.Error("an activity that still exists on Strava was deleted")
	}
	e.strava.deleteActivity(2000)
	send(activityEvent("delete", 2000, 7))
	if _, ok := getActivity(2000); ok {
		t.Error("the deleted activity wasn't deleted")
	}

	// Revoking access disconnects Strava, once Strava rejects the token.
	deauthorize := `{"aspect_type": "update", "event_time": 1757232000, "object_id": 7, "object_type": "athlete",
		"owner_id": 7, "subscription_id": 100, "updates": {"authorized": "false"}}`
	send(deauthorize)
	if _, err := e.activities.GetStravaConnection(ctx, e.userID); err != nil {
		t.Fatalf("an authorized connection was removed: %v", err)
	}
	e.strava.mu.Lock()
	e.strava.revoked = true
	e.strava.mu.Unlock()
	send(deauthorize)
	if _, err := e.activities.GetStravaConnection(ctx, e.userID); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("got %v, want the connection removed", err)
	}

	rec := httptest.NewRecorder()
	webhook.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/strava/webhook", strings.NewReader("not json")))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d for an invalid event, want 400", rec.Code)
	}
}
