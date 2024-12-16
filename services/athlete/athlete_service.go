package athlete

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/athletes"
	athletesv1 "github.com/catzkorn/trail-tools/gen/athletes/v1"
	"github.com/catzkorn/trail-tools/gen/athletes/v1/athletesv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ athletesv1connect.AthleteServiceHandler = (*Service)(nil)

type Directory interface {
	AddAthlete(ctx context.Context, name string) (athletes.Athlete, error)
	AddActivity(ctx context.Context, activity *athletes.AddActivityParams) (athletes.Activity, error)
	AddMeasure(ctx context.Context, measure *athletes.AddMeasureParams) (athletes.BloodLactateMeasure, error)
}

// Service implements API handlers for the athlete service.
type Service struct {
	log       *slog.Logger
	directory Directory
}

// NewService creates a new Service from the provided logger and directory.
func NewService(log *slog.Logger, directory Directory) *Service {
	return &Service{
		log:       log,
		directory: directory,
	}
}

func (s *Service) CreateAthlete(ctx context.Context, req *connect.Request[athletesv1.CreateAthleteRequest]) (*connect.Response[athletesv1.CreateAthleteResponse], error) {
	if req.Msg.Name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("empty athlete name"))
	}
	athlete, err := s.directory.AddAthlete(ctx, req.Msg.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add athlete: %w", err))
	}
	resp := &athletesv1.CreateAthleteResponse{
		Athlete: &athletesv1.Athlete{
			Name:       req.Msg.Name,
			CreateTime: timestamppb.New(athlete.CreateTime.Time),
		},
	}
	uuid, err := athlete.ID.Value()
	resp.Athlete.Id = uuid.(string)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to assign athlete ID: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (s *Service) CreateActivity(ctx context.Context, req *connect.Request[athletesv1.CreateActivityRequest]) (*connect.Response[athletesv1.CreateActivityResponse], error) {
	activityParam := &athletes.AddActivityParams{
		Name: req.Msg.Name,
	}
	err := activityParam.AthleteID.Scan(req.Msg.AthleteId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid athlete ID: %w", err))
	}
	activity, err := s.directory.AddActivity(ctx, activityParam)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add activity: %w", err))
	}
	resp := &athletesv1.CreateActivityResponse{
		Activity: &athletesv1.Activity{
			Name:       req.Msg.Name,
			CreateTime: timestamppb.New(activity.CreateTime.Time),
		},
	}
	uuid, err := activity.ID.Value()
	resp.Activity.Id = uuid.(string)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to assign activity ID: %w", err))
	}
	return connect.NewResponse(resp), nil
}

func (s *Service) CreateBloodLactateMeasure(ctx context.Context, req *connect.Request[athletesv1.CreateBloodLactateMeasureRequest]) (*connect.Response[athletesv1.CreateBloodLactateMeasureResponse], error) {
	if req.Msg.HeartRateBpm <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid heart rate: %d", req.Msg.HeartRateBpm))
	}
	measureParam := &athletes.AddMeasureParams{
		HeartRateBpm: req.Msg.HeartRateBpm,
	}
	if err := measureParam.ActivityID.Scan(req.Msg.ActivityId); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid activity ID: %w", err))
	}
	if err := measureParam.MmolPerLiter.Scan(req.Msg.MmolPerLiter); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid mmol per liter: %w", err))
	}
	measure, err := s.directory.AddMeasure(ctx, measureParam)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to add measure: %w", err))
	}
	mmol, err := measure.MmolPerLiter.Value()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get mmol per liter from DB: %w", err))
	}
	resp := &athletesv1.CreateBloodLactateMeasureResponse{
		BloodLactateMeasure: &athletesv1.BloodLactateMeasure{
			MmolPerLiter: mmol.(string),
			HeartRateBpm: int32(measure.HeartRateBpm),
		},
	}
	uuid, err := measure.ID.Value()
	resp.BloodLactateMeasure.Id = uuid.(string)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to assign measure ID: %w", err))
	}
	return connect.NewResponse(resp), nil
}
