package athlete

import (
	"context"
	"errors"

	"connectrpc.com/connect/v2"
	"github.com/catzkorn/trail-tools/internal/authn"
	athletesv1 "github.com/catzkorn/trail-tools/internal/gen/athletes/v1"
	"github.com/catzkorn/trail-tools/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) CreateAthlete(ctx context.Context, req *athletesv1.CreateAthleteRequest) (*athletesv1.CreateAthleteResponse, error) {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated")
	}
	if req.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "empty athlete name")
	}
	athlete, err := s.athletes.AddAthlete(ctx, req.GetName(), user.ID())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "failed to add athlete").WithCause(err)
	}
	return athletesv1.CreateAthleteResponse_builder{
		Athlete: athletesv1.Athlete_builder{
			Name:       req.GetName(),
			CreateTime: timestamppb.New(athlete.CreateTime.Time),
			Id:         store.UUIDToString(athlete.ID),
		}.Build(),
	}.Build(), nil
}

func (s *Service) ListAthletes(ctx context.Context, req *athletesv1.ListAthletesRequest) (*athletesv1.ListAthletesResponse, error) {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated")
	}
	athletes, err := s.athletes.ListAthletesForUser(ctx, user.ID())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "failed to list athletes").WithCause(err)
	}
	var athleteResponses []*athletesv1.Athlete
	for _, athlete := range athletes {
		athleteResponses = append(athleteResponses, athletesv1.Athlete_builder{
			Name:       athlete.Name,
			CreateTime: timestamppb.New(athlete.CreateTime.Time),
			Id:         store.UUIDToString(athlete.ID),
		}.Build())
	}
	return athletesv1.ListAthletesResponse_builder{
		Athletes: athleteResponses,
	}.Build(), nil
}

func (s *Service) DeleteAthlete(ctx context.Context, req *athletesv1.DeleteAthleteRequest) (*athletesv1.DeleteAthleteResponse, error) {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated")
	}
	athleteID, err := store.StringToUUID(req.GetId())
	if err != nil {
		return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid athlete ID: %v", err).WithCause(err)
	}
	if err := s.athletes.DeleteAthleteForUser(ctx, user.ID(), athleteID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "athlete not found")
		}
		return nil, connect.NewError(connect.CodeInternal, "failed to delete athlete").WithCause(err)
	}
	return athletesv1.DeleteAthleteResponse_builder{}.Build(), nil
}
