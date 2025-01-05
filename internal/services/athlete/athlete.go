package athlete

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	athletesv1 "github.com/catzkorn/trail-tools/internal/gen/athletes/v1"
	"github.com/catzkorn/trail-tools/internal/oidc"
	"github.com/catzkorn/trail-tools/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) CreateAthlete(ctx context.Context, req *connect.Request[athletesv1.CreateAthleteRequest]) (*connect.Response[athletesv1.CreateAthleteResponse], error) {
	userInfo, ok := oidc.GetUserInfo(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("empty athlete name"))
	}
	user, err := s.users.GetOIDCUser(ctx, userInfo.Subject)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get user: %w", err))
	}
	athlete, err := s.athletes.AddAthlete(ctx, req.Msg.GetName(), user.ID)
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
