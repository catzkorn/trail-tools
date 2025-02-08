package user

import (
	"log/slog"

	"github.com/catzkorn/trail-tools/internal/gen/users/v1/usersv1connect"
)

var _ usersv1connect.UserServiceHandler = (*Service)(nil)

// Service implements API handlers for the athlete service.
type Service struct {
	log *slog.Logger
}

// NewService creates a new Service from the provided logger and directory.
func NewService(log *slog.Logger) *Service {
	return &Service{
		log: log,
	}
}
