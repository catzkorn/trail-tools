package athlete

import (
	"context"
	"log/slog"

	"github.com/catzkorn/trail-tools/athletes"
	"github.com/catzkorn/trail-tools/gen/athletes/v1/athletesv1connect"
	"github.com/catzkorn/trail-tools/users"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

var _ athletesv1connect.AthleteServiceHandler = (*Service)(nil)

type AthleteRepository interface {
	AddAthlete(ctx context.Context, name string, userID pgtype.UUID) (athletes.Athlete, error)
	AddActivity(ctx context.Context, name string, athleteID pgtype.UUID) (athletes.Activity, error)
	AddMeasure(ctx context.Context, activityID pgtype.UUID, mmolPerLiter decimal.Decimal, heartRateBPM int32) (athletes.BloodLactateMeasure, error)
}

type UserRepository interface {
	GetUser(ctx context.Context, oidcSubject string) (users.User, error)
}

// Service implements API handlers for the athlete service.
type Service struct {
	log      *slog.Logger
	users    UserRepository
	athletes AthleteRepository
}

// NewService creates a new Service from the provided logger and directory.
func NewService(log *slog.Logger, users UserRepository, athletes AthleteRepository) *Service {
	return &Service{
		log:      log,
		users:    users,
		athletes: athletes,
	}
}
