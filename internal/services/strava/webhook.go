package strava

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/johanbrandhorst/trail-tools/internal/activities"
	"github.com/johanbrandhorst/trail-tools/internal/errreport"
	"github.com/johanbrandhorst/trail-tools/internal/store"
	"github.com/johanbrandhorst/trail-tools/internal/strava"
)

// WebhookPath is the path Strava sends webhook events to.
const WebhookPath = "/strava/webhook"

const (
	// webhookBudget is how long an event is handled for before responding.
	// Strava expects a response within two seconds, and sends the event
	// again otherwise.
	webhookBudget = 1500 * time.Millisecond
	// maxEventBytes limits the size of webhook events, which are small.
	maxEventBytes = 64 << 10
	// subscribeTimeout bounds creating the push subscription, during which
	// Strava calls the webhook.
	subscribeTimeout = 30 * time.Second
)

// WebhookRepository finds the connections webhook events are for.
type WebhookRepository interface {
	ListStravaConnectionsByAthlete(ctx context.Context, athleteID int64) ([]*activities.StravaConnection, error)
	DeleteStravaConnection(ctx context.Context, userID pgtype.UUID) error
}

// SubscriptionClient manages the application's push subscription.
type SubscriptionClient interface {
	ListPushSubscriptions(ctx context.Context) ([]*strava.PushSubscription, error)
	CreatePushSubscription(ctx context.Context, callbackURL, verifyToken string) (*strava.PushSubscription, error)
	DeletePushSubscription(ctx context.Context, id int64) error
}

// Webhook receives Strava's webhook events, so new and changed activities
// are synced within seconds instead of at the next sync. Strava doesn't
// sign events, so an event is only taken as a hint: the activity is fetched
// from Strava with the athlete's token, and deleted only if Strava no longer
// has it, and a connection is removed only if Strava rejects its token.
type Webhook struct {
	log         *slog.Logger
	report      *errreport.Reporter
	repo        WebhookRepository
	syncer      *Syncer
	client      SubscriptionClient
	callbackURL string
	verifyToken string
}

// NewWebhook creates a new Webhook, served at callbackURL. verifyToken is
// sent by Strava when validating the push subscription.
func NewWebhook(log *slog.Logger, report *errreport.Reporter, repo WebhookRepository, syncer *Syncer, client SubscriptionClient, callbackURL, verifyToken string) *Webhook {
	return &Webhook{
		log:         log,
		report:      report,
		repo:        repo,
		syncer:      syncer,
		client:      client,
		callbackURL: callbackURL,
		verifyToken: verifyToken,
	}
}

// Subscribe makes sure Strava sends events to the webhook, replacing a
// subscription to another URL, as Strava allows one per application. Strava
// validates the webhook while creating the subscription, so the server must
// be serving it.
func (h *Webhook) Subscribe(ctx context.Context) error {
	if err := checkPublic(h.callbackURL); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, subscribeTimeout)
	defer cancel()
	subs, err := h.client.ListPushSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to list push subscriptions: %w", err)
	}
	for _, sub := range subs {
		if sub.CallbackURL == h.callbackURL {
			h.log.InfoContext(ctx, "Subscribed to Strava webhook events", slog.String("callback_url", h.callbackURL))
			return nil
		}
		h.log.WarnContext(ctx, "Replacing the Strava push subscription to another URL",
			slog.Int64("subscription_id", sub.ID), slog.String("callback_url", sub.CallbackURL))
		if err := h.client.DeletePushSubscription(ctx, sub.ID); err != nil && !errors.Is(err, strava.ErrNotFound) {
			return fmt.Errorf("failed to delete push subscription %d: %w", sub.ID, err)
		}
	}
	sub, err := h.client.CreatePushSubscription(ctx, h.callbackURL, h.verifyToken)
	if err != nil {
		return fmt.Errorf("failed to create push subscription: %w", err)
	}
	h.log.InfoContext(ctx, "Created Strava push subscription",
		slog.Int64("subscription_id", sub.ID), slog.String("callback_url", h.callbackURL))
	return nil
}

// checkPublic returns an error if Strava can't reach the URL.
func checkPublic(callbackURL string) error {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if host == "localhost" || ip != nil && (ip.IsLoopback() || ip.IsUnspecified() || ip.IsPrivate()) {
		return fmt.Errorf("%s isn't reachable by Strava, so webhook events need a public URL", callbackURL)
	}
	return nil
}

func (h *Webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleValidation(w, r)
	case http.MethodPost:
		h.handleEvent(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleValidation answers Strava's validation of the callback, while
// creating the push subscription, by echoing the challenge.
func (h *Webhook) handleValidation(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("hub.mode") != "subscribe" || subtle.ConstantTimeCompare([]byte(q.Get("hub.verify_token")), []byte(h.verifyToken)) != 1 {
		http.Error(w, "invalid subscription validation", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"hub.challenge": q.Get("hub.challenge")})
}

// event is a webhook event, see https://developers.strava.com/docs/webhooks/.
type event struct {
	// "activity" or "athlete".
	ObjectType string `json:"object_type"`
	// The ID of the activity or athlete.
	ObjectID int64 `json:"object_id"`
	// "create", "update" or "delete".
	AspectType string `json:"aspect_type"`
	// The changed fields of an update. Athletes revoking access are sent as
	// an update with "authorized": "false".
	Updates map[string]any `json:"updates"`
	// The athlete who owns the object.
	OwnerID int64 `json:"owner_id"`
}

func (h *Webhook) handleEvent(w http.ResponseWriter, r *http.Request) {
	var e event
	if err := json.NewDecoder(io.LimitReader(r.Body, maxEventBytes)).Decode(&e); err != nil {
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}
	// Strava hangs up after two seconds, which cancels the request's
	// context. Keep going for the budget anyway, so the work isn't lost.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), webhookBudget)
	defer cancel()
	attrs := []slog.Attr{
		slog.String("object_type", e.ObjectType), slog.Int64("object_id", e.ObjectID),
		slog.String("aspect_type", e.AspectType), slog.Int64("owner_id", e.OwnerID),
	}
	log := slog.New(h.log.Handler().WithAttrs(attrs))
	report := h.report.With(attrs...)
	conns, err := h.repo.ListStravaConnectionsByAthlete(ctx, e.OwnerID)
	if err != nil {
		// Strava sends the event again.
		report.Report(ctx, "Failed to find the Strava connection of a webhook event", err)
		http.Error(w, "failed to handle event", http.StatusInternalServerError)
		return
	}
	if len(conns) == 0 {
		log.DebugContext(ctx, "Ignoring a Strava webhook event of an unknown athlete")
	}
	for _, conn := range conns {
		switch {
		case e.ObjectType == "athlete" && e.AspectType == "update" && fmt.Sprint(e.Updates["authorized"]) == "false":
			h.deauthorize(ctx, log, report, conn)
		case e.ObjectType == "activity":
			h.syncActivity(ctx, log, report, conn, e.ObjectID)
		}
	}
	w.WriteHeader(http.StatusOK)
}

// syncActivity syncs an activity that was created, updated or deleted. If
// that fails or takes too long, the connection is synced soon instead.
func (h *Webhook) syncActivity(ctx context.Context, log *slog.Logger, report *errreport.Reporter, conn *activities.StravaConnection, stravaID int64) {
	user := slog.String("user_id", store.UUIDToString(conn.UserID))
	log, report = log.With(user), report.With(user)
	err := h.syncer.SyncActivity(ctx, conn.UserID, stravaID)
	switch {
	case err == nil:
		log.InfoContext(ctx, "Synced activity from a Strava webhook event")
		return
	case errors.Is(err, errRateLimited):
		// The connection is recorded as rate limited, and syncs once the
		// limit resets.
		log.InfoContext(ctx, "Strava webhook event deferred by the rate limit")
		return
	case errors.Is(err, store.ErrNotFound):
		// Strava was disconnected meanwhile.
		return
	case ctx.Err() != nil:
		log.InfoContext(ctx, "Syncing an activity from a Strava webhook event took too long, syncing later")
	default:
		report.Report(ctx, "Failed to sync an activity from a Strava webhook event, syncing later", err)
	}
	stateCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := h.syncer.SyncSoon(stateCtx, conn.UserID); err != nil {
		report.Report(ctx, "Failed to schedule a Strava sync", err)
	}
}

// deauthorize removes the connection if the athlete revoked access.
func (h *Webhook) deauthorize(ctx context.Context, log *slog.Logger, report *errreport.Reporter, conn *activities.StravaConnection) {
	user := slog.String("user_id", store.UUIDToString(conn.UserID))
	log, report = log.With(user), report.With(user)
	authorized, err := h.syncer.Authorized(ctx, conn)
	switch {
	case err != nil:
		report.Report(ctx, "Failed to check the Strava authorization of a webhook event", err)
	case authorized:
		log.WarnContext(ctx, "Ignoring a Strava deauthorization event of an athlete who is still authorized")
	default:
		if err := h.repo.DeleteStravaConnection(ctx, conn.UserID); err != nil && !errors.Is(err, store.ErrNotFound) {
			report.Report(ctx, "Failed to disconnect Strava", err)
			return
		}
		log.InfoContext(ctx, "Disconnected Strava, as the athlete revoked access")
	}
}
