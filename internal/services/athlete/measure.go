package athlete

import (
	"context"

	"connectrpc.com/connect/v2"
	"github.com/catzkorn/trail-tools/internal/authn"
	athletesv1 "github.com/catzkorn/trail-tools/internal/gen/athletes/v1"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) CreateBloodLactateMeasure(ctx context.Context, req *athletesv1.CreateBloodLactateMeasureRequest) (*athletesv1.CreateBloodLactateMeasureResponse, error) {
	_, ok := authn.GetUser(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "unauthenticated")
	}
	if req.GetHeartRateBpm() <= 0 {
		return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid heart rate: %d", req.GetHeartRateBpm())
	}
	activityID, err := store.StringToUUID(req.GetActivityId())
	if err != nil {
		return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid activity ID: %v", err).WithCause(err)
	}
	mmol, err := decimal.NewFromString(req.GetMmolPerLiter())
	if err != nil {
		return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid mmol per liter: %v", err).WithCause(err)
	}
	if mmol.LessThan(decimal.Zero) {
		return nil, connect.Errorf(connect.CodeInvalidArgument, "mmol per liter must be positive: %s", req.GetMmolPerLiter())
	}
	measure, err := s.athletes.AddMeasure(ctx, activityID, mmol, req.GetHeartRateBpm())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "failed to add measure").WithCause(err)
	}
	return athletesv1.CreateBloodLactateMeasureResponse_builder{
		BloodLactateMeasure: athletesv1.BloodLactateMeasure_builder{
			MmolPerLiter: measure.MmolPerLiter.String(),
			HeartRateBpm: int32(measure.HeartRateBpm),
			CreateTime:   timestamppb.New(measure.CreateTime.Time),
			Id:           store.UUIDToString(measure.ID),
			ActivityId:   store.UUIDToString(measure.ActivityID),
		}.Build(),
	}.Build(), nil
}
