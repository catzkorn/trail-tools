package user

import (
	"context"
	"log/slog"

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

// Service implements API handlers for the user service.
type Service struct {
	log    *slog.Logger
	tokens APITokenRepository
}

// NewService creates a new Service from the provided logger and repository.
func NewService(log *slog.Logger, tokens APITokenRepository) *Service {
	return &Service{
		log:    log,
		tokens: tokens,
	}
}
