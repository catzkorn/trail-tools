// Package strava implements the StravaService, the Strava OAuth flow and
// the background sync of activities.
package strava

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect/v2"
	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/authn"
	stravav1 "github.com/catzkorn/trail-tools/internal/gen/strava/v1"
	"github.com/catzkorn/trail-tools/internal/gen/strava/v1/stravav1connect"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"gitlab.com/greyxor/slogor"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ stravav1connect.StravaServiceHandler = (*Service)(nil)

type ConnectionRepository interface {
	GetStravaConnection(ctx context.Context, userID pgtype.UUID) (*activities.StravaConnection, error)
	UpsertStravaConnection(ctx context.Context, conn *activities.StravaConnection) (*activities.StravaConnection, error)
	DeleteStravaConnection(ctx context.Context, userID pgtype.UUID) error
	CountActivities(ctx context.Context, userID pgtype.UUID) (total, pendingStreams int64, err error)
}

// Service implements API handlers for the Strava service. If the server is
// not configured with Strava credentials, the syncer and OAuth client are nil.
type Service struct {
	log    *slog.Logger
	repo   ConnectionRepository
	syncer *Syncer
	oauth  *OAuth
}

// NewService creates a new Service. syncer and oauth may be nil if Strava is
// not configured.
func NewService(log *slog.Logger, repo ConnectionRepository, syncer *Syncer, oauth *OAuth) *Service {
	return &Service{
		log:    log,
		repo:   repo,
		syncer: syncer,
		oauth:  oauth,
	}
}

func (s *Service) GetStravaConnection(ctx context.Context, req *stravav1.GetStravaConnectionRequest) (*stravav1.GetStravaConnectionResponse, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	conn, err := s.connection(ctx, userID)
	if err != nil {
		return nil, err
	}
	return stravav1.GetStravaConnectionResponse_builder{Connection: conn}.Build(), nil
}

func (s *Service) SyncStrava(ctx context.Context, req *stravav1.SyncStravaRequest) (*stravav1.SyncStravaResponse, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	if s.syncer == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, "the server is not configured with Strava API credentials")
	}
	if _, err := s.repo.GetStravaConnection(ctx, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, "no Strava account is connected; connect one in the browser at /strava/connect")
		}
		return nil, s.internalError(ctx, "failed to get Strava connection", err)
	}
	if s.syncer.Background() {
		s.syncer.Start(userID, req.GetFull())
	} else {
		// Without background work, sync for a while within the request.
		// Scheduled syncs continue from where this one stops.
		syncCtx, cancel := context.WithTimeout(ctx, inlineSyncBudget)
		s.syncer.SyncOnce(syncCtx, userID, req.GetFull())
		cancel()
	}
	conn, err := s.connection(ctx, userID)
	if err != nil {
		return nil, err
	}
	return stravav1.SyncStravaResponse_builder{Connection: conn}.Build(), nil
}

func (s *Service) DisconnectStrava(ctx context.Context, req *stravav1.DisconnectStravaRequest) (*stravav1.DisconnectStravaResponse, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	if err := s.repo.DeleteStravaConnection(ctx, userID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "no Strava account is connected")
		}
		return nil, s.internalError(ctx, "failed to disconnect Strava", err)
	}
	return stravav1.DisconnectStravaResponse_builder{}.Build(), nil
}

func (s *Service) connection(ctx context.Context, userID pgtype.UUID) (*stravav1.StravaConnection, error) {
	b := stravav1.StravaConnection_builder{
		Configured: s.oauth != nil,
		SyncState:  stravav1.SyncState_SYNC_STATE_IDLE,
	}
	conn, err := s.repo.GetStravaConnection(ctx, userID)
	switch {
	case errors.Is(err, store.ErrNotFound):
	case err != nil:
		return nil, s.internalError(ctx, "failed to get Strava connection", err)
	default:
		b.Connected = true
		b.AthleteName = conn.AthleteName
		b.AthleteId = fmt.Sprint(conn.AthleteID)
		b.SyncState = syncStateToProto(conn.SyncState)
		b.SyncMessage = conn.SyncMessage
		if !conn.LastSyncTime.IsZero() {
			b.LastSyncTime = timestamppb.New(conn.LastSyncTime)
		}
		if !conn.ResumeTime.IsZero() && conn.SyncState == activities.SyncStateRateLimited {
			b.ResumeTime = timestamppb.New(conn.ResumeTime)
		}
		// A sync recorded as running may have been interrupted by a restart
		// and not resumed yet.
		if conn.SyncState == activities.SyncStateRunning && (s.syncer == nil || !s.syncer.Running(userID)) {
			b.SyncState = stravav1.SyncState_SYNC_STATE_IDLE
			b.SyncMessage = "The last sync was interrupted and will resume shortly"
		}
	}
	total, pending, err := s.repo.CountActivities(ctx, userID)
	if err != nil {
		return nil, s.internalError(ctx, "failed to count activities", err)
	}
	b.ActivityCount = int32(total)
	b.PendingStreamCount = int32(pending)
	return b.Build(), nil
}

func syncStateToProto(state activities.SyncState) stravav1.SyncState {
	switch state {
	case activities.SyncStateRunning:
		return stravav1.SyncState_SYNC_STATE_RUNNING
	case activities.SyncStateRateLimited:
		return stravav1.SyncState_SYNC_STATE_RATE_LIMITED
	case activities.SyncStateFailed:
		return stravav1.SyncState_SYNC_STATE_FAILED
	default:
		return stravav1.SyncState_SYNC_STATE_IDLE
	}
}

func (s *Service) internalError(ctx context.Context, msg string, err error) error {
	s.log.ErrorContext(ctx, msg, slogor.Err(err))
	return connect.NewError(connect.CodeInternal, msg).WithCause(err)
}

// inlineSyncBudget is how long SyncStrava syncs for when syncs don't run in
// the background.
const inlineSyncBudget = 45 * time.Second

// timeNow is overridden in tests.
var timeNow = time.Now
