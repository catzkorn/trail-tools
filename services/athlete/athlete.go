package athlete

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	athletesv1 "github.com/catzkorn/trail-tools/gen/athletes/v1"
	"github.com/catzkorn/trail-tools/oidc"
	"github.com/catzkorn/trail-tools/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) CreateAthlete(ctx context.Context, req *connect.Request[athletesv1.CreateAthleteRequest]) (*connect.Response[athletesv1.CreateAthleteResponse], error) {
	userInfo := oidc.GetAuthDetails(ctx)
	if userInfo == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("empty athlete name"))
	}
	user, err := s.users.GetUser(ctx, userInfo.Subject)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get user: %w", err))
	}
	athlete, err := s.athletes.AddAthlete(ctx, req.Msg.Name, user.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add athlete: %w", err))
	}
	resp := &athletesv1.CreateAthleteResponse{
		Athlete: &athletesv1.Athlete{
			Name:       req.Msg.Name,
			CreateTime: timestamppb.New(athlete.CreateTime.Time),
			Id:         store.UUIDToString(athlete.ID),
		},
	}
	return connect.NewResponse(resp), nil
}
