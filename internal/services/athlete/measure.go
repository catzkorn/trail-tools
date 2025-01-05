package athlete

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	athletesv1 "github.com/catzkorn/trail-tools/internal/gen/athletes/v1"
	"github.com/catzkorn/trail-tools/internal/oidc"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) CreateBloodLactateMeasure(ctx context.Context, req *connect.Request[athletesv1.CreateBloodLactateMeasureRequest]) (*connect.Response[athletesv1.CreateBloodLactateMeasureResponse], error) {
	_, ok := oidc.GetUserInfo(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("unauthenticated"))
	}
	if req.Msg.GetHeartRateBpm() <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid heart rate: %d", req.Msg.GetHeartRateBpm()))
	}
	activityID, err := store.StringToUUID(req.Msg.GetActivityId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid activity ID: %w", err))
	}
	mmol, err := decimal.NewFromString(req.Msg.GetMmolPerLiter())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid mmol per liter: %w", err))
	}
	if mmol.LessThan(decimal.Zero) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("mmol per liter must be positive: %s", req.Msg.GetMmolPerLiter()))
	}
	measure, err := s.athletes.AddMeasure(ctx, activityID, mmol, req.Msg.GetHeartRateBpm())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add measure: %w", err))
	}
	return connect.NewResponse(athletesv1.CreateBloodLactateMeasureResponse_builder{
		BloodLactateMeasure: athletesv1.BloodLactateMeasure_builder{
			MmolPerLiter: measure.MmolPerLiter.String(),
			HeartRateBpm: int32(measure.HeartRateBpm),
			CreateTime:   timestamppb.New(measure.CreateTime.Time),
			Id:           store.UUIDToString(measure.ID),
			ActivityId:   store.UUIDToString(measure.ActivityID),
		}.Build(),
	}.Build()), nil
}
