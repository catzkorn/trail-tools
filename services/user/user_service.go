package user

import (
	"context"
	"log/slog"

	"github.com/catzkorn/trail-tools/gen/users/v1/usersv1connect"
	"github.com/catzkorn/trail-tools/users"
)

var _ usersv1connect.UserServiceHandler = (*Service)(nil)

type UserRepository interface {
	GetUser(ctx context.Context, oidcSubject string) (users.User, error)
}

// Service implements API handlers for the athlete service.
type Service struct {
	log   *slog.Logger
	users UserRepository
}

// NewService creates a new Service from the provided logger and directory.
func NewService(log *slog.Logger, users UserRepository) *Service {
	return &Service{
		log:   log,
		users: users,
	}
}
