package user

import (
	"context"
	"log/slog"

	"github.com/catzkorn/trail-tools/internal/blob"
	"github.com/catzkorn/trail-tools/internal/errreport"
	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/internal/users"
	"github.com/jackc/pgx/v5/pgtype"
)

var _ usersv1connect.UserServiceHandler = (*Service)(nil)

type APITokenRepository interface {
	CreateAPIToken(ctx context.Context, userID pgtype.UUID, name string) (*users.APIToken, string, error)
	ListAPITokens(ctx context.Context, userID pgtype.UUID) ([]*users.APIToken, error)
	DeleteAPIToken(ctx context.Context, userID pgtype.UUID, tokenID pgtype.UUID) error
}

type AvatarRepository interface {
	SetAvatarKey(ctx context.Context, userID pgtype.UUID, key string) (string, error)
}

// Service implements API handlers for the user service.
type Service struct {
	log     *slog.Logger
	report  *errreport.Reporter
	tokens  APITokenRepository
	avatars AvatarRepository
	blobs   blob.Store
}

// NewService creates a new Service from the provided logger, repositories
// and the blob store avatar images are stored in.
func NewService(log *slog.Logger, report *errreport.Reporter, tokens APITokenRepository, avatars AvatarRepository, blobs blob.Store) *Service {
	return &Service{
		log:     log,
		report:  report,
		tokens:  tokens,
		avatars: avatars,
		blobs:   blobs,
	}
}
