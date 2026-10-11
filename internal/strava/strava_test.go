package strava

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimit(t *testing.T) {
	t.Parallel()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-ReadRateLimit-Limit", "100,1000")
		w.Header().Set("X-ReadRateLimit-Usage", fmt.Sprintf("%d,%d", 97+calls, 500+calls))
		fmt.Fprint(w, `[{"id": 1, "name": "Morning Run", "sport_type": "TrailRun", "start_date": "2026-09-07T07:00:00Z",
			"start_date_local": "2026-09-07T08:00:00Z", "timezone": "(GMT+01:00) Europe/London", "average_cadence": 85}]`)
	}))
	t.Cleanup(srv.Close)
	client := New(Config{BaseURL: srv.URL})
	now := time.Date(2026, 9, 7, 10, 3, 0, 0, time.UTC)
	client.nowFn = func() time.Time { return now }
	ctx := context.Background()

	activities, err := client.ListActivities(ctx, "token", time.Time{}, 1, 10)
	if err != nil {
		t.Fatalf("ListActivities: %v", err)
	}
	a := activities[0]
	if a.TimezoneName() != "Europe/London" || !a.LocalDate().Equal(time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)) || !a.IsFootSport() || len(a.Raw) == 0 {
		t.Errorf("unexpected activity %+v", a)
	}
	// 98 of 100 requests used: the next request would be too close to the
	// limit, so it is rejected until the next 15 minute window.
	_, err = client.ListActivities(ctx, "token", time.Time{}, 2, 10)
	var rateErr *RateLimitError
	if !errors.As(err, &rateErr) || !rateErr.ResetAt.Equal(time.Date(2026, 9, 7, 10, 15, 0, 0, time.UTC)) {
		t.Fatalf("got %v, want rate limit until 10:15", err)
	}
	if calls != 1 {
		t.Errorf("rate limited request should not reach Strava, got %d calls", calls)
	}
	now = now.Add(15 * time.Minute)
	if _, err := client.ListActivities(ctx, "token", time.Time{}, 2, 10); err != nil {
		t.Fatalf("request after reset: %v", err)
	}
	if _, err := New(Config{BaseURL: srv.URL}).ListActivities(ctx, "bad", time.Time{}, 2, 10); !errors.Is(err, ErrUnauthorized) {
		t.Errorf("got %v, want ErrUnauthorized", err)
	}
}

func TestTooManyRequests(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "200,2000")
		w.Header().Set("X-RateLimit-Usage", "10,2000")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	client := New(Config{BaseURL: srv.URL})
	now := time.Date(2026, 9, 7, 10, 3, 0, 0, time.UTC)
	client.nowFn = func() time.Time { return now }
	_, err := client.GetStreams(context.Background(), "token", 1)
	var rateErr *RateLimitError
	// The daily limit is used up, so wait until midnight UTC.
	if !errors.As(err, &rateErr) || !rateErr.ResetAt.Equal(time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("got %v, want rate limit until midnight", err)
	}
}

func TestGetStreams(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/activities/2/streams" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"latlng": {"data": [[1.5, 2.5], [1.6, 2.6]]}, "altitude": {"data": [10, 11]}, "time": {"data": [0, 5]}}`)
	}))
	t.Cleanup(srv.Close)
	client := New(Config{BaseURL: srv.URL})
	streams, err := client.GetStreams(context.Background(), "token", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(streams.LatLng) != 2 || streams.LatLng[1][1] != 2.6 || streams.Altitude[1] != 11 || streams.Time[1] != 5 || streams.Heartrate != nil {
		t.Errorf("unexpected streams %+v", streams)
	}
	if _, err := client.GetStreams(context.Background(), "token", 2); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}
