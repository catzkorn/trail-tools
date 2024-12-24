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

func (s *Service) CreateActivity(ctx context.Context, req *connect.Request[athletesv1.CreateActivityRequest]) (*connect.Response[athletesv1.CreateActivityResponse], error) {
	userInfo := oidc.GetAuthDetails(ctx)
	if userInfo == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	athleteID, err := store.StringToUUID(req.Msg.AthleteId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid athlete ID: %w", err))
	}
	activity, err := s.athletes.AddActivity(ctx, req.Msg.Name, athleteID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add activity: %w", err))
	}
	resp := &athletesv1.CreateActivityResponse{
		Activity: &athletesv1.Activity{
			Id:         store.UUIDToString(activity.ID),
			AthleteId:  store.UUIDToString(activity.AthleteID),
			Name:       req.Msg.Name,
			CreateTime: timestamppb.New(activity.CreateTime.Time),
		},
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to assign activity ID: %w", err))
	}
	return connect.NewResponse(resp), nil
}
