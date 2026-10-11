package athlete

import (
	"context"

	"connectrpc.com/connect/v2"
	athletesv1 "github.com/catzkorn/trail-tools/internal/gen/athletes/v1"
	"github.com/catzkorn/trail-tools/internal/oidc"
	"github.com/catzkorn/trail-tools/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) CreateActivity(ctx context.Context, req *athletesv1.CreateActivityRequest) (*athletesv1.CreateActivityResponse, error) {
	_, ok := oidc.GetUserInfo(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated")
	}
	athleteID, err := store.StringToUUID(req.GetAthleteId())
	if err != nil {
		return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid athlete ID: %v", err).WithCause(err)
	}
	activity, err := s.athletes.AddActivity(ctx, req.GetName(), athleteID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "failed to add activity").WithCause(err)
	}
	return athletesv1.CreateActivityResponse_builder{
		Activity: athletesv1.Activity_builder{
			Id:         store.UUIDToString(activity.ID),
			AthleteId:  store.UUIDToString(activity.AthleteID),
			Name:       req.GetName(),
			CreateTime: timestamppb.New(activity.CreateTime.Time),
		}.Build(),
	}.Build(), nil
}
