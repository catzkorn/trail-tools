package athlete

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/authn"
	athletesv1 "github.com/catzkorn/trail-tools/internal/gen/athletes/v1"
	"github.com/catzkorn/trail-tools/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) CreateAthlete(ctx context.Context, req *connect.Request[athletesv1.CreateAthleteRequest]) (*connect.Response[athletesv1.CreateAthleteResponse], error) {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("empty athlete name"))
	}
	athlete, err := s.athletes.AddAthlete(ctx, req.Msg.GetName(), user.ID())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add athlete: %w", err))
	}
	return connect.NewResponse(athletesv1.CreateAthleteResponse_builder{
		Athlete: athletesv1.Athlete_builder{
			Name:       req.Msg.GetName(),
			CreateTime: timestamppb.New(athlete.CreateTime.Time),
			Id:         store.UUIDToString(athlete.ID),
		}.Build(),
	}.Build()), nil
}

func (s *Service) ListAthletes(ctx context.Context, req *connect.Request[athletesv1.ListAthletesRequest]) (*connect.Response[athletesv1.ListAthletesResponse], error) {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	athletes, err := s.athletes.ListAthletesForUser(ctx, user.ID())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list athletes: %w", err))
	}
	var athleteResponses []*athletesv1.Athlete
	for _, athlete := range athletes {
		athleteResponses = append(athleteResponses, athletesv1.Athlete_builder{
			Name:       athlete.Name,
			CreateTime: timestamppb.New(athlete.CreateTime.Time),
			Id:         store.UUIDToString(athlete.ID),
		}.Build())
	}
	return connect.NewResponse(athletesv1.ListAthletesResponse_builder{
		Athletes: athleteResponses,
	}.Build()), nil
}

func (s *Service) DeleteAthlete(ctx context.Context, req *connect.Request[athletesv1.DeleteAthleteRequest]) (*connect.Response[athletesv1.DeleteAthleteResponse], error) {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	athleteID, err := store.StringToUUID(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid athlete ID: %w", err))
	}
	if err := s.athletes.DeleteAthleteForUser(ctx, user.ID(), athleteID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("athlete not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to delete athlete: %w", err))
	}
	return connect.NewResponse(athletesv1.DeleteAthleteResponse_builder{}.Build()), nil
}
