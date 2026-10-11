// Package raceplan implements the RacePlanService.
package raceplan

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/catzkorn/trail-tools/internal/authn"
	racesv1 "github.com/catzkorn/trail-tools/internal/gen/races/v1"
	"github.com/catzkorn/trail-tools/internal/gen/races/v1/racesv1connect"
	routesv1 "github.com/catzkorn/trail-tools/internal/gen/routes/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/protomask"
	"github.com/catzkorn/trail-tools/internal/races"
	"github.com/catzkorn/trail-tools/internal/routes"
	"github.com/catzkorn/trail-tools/internal/services/race"
	"github.com/catzkorn/trail-tools/internal/services/route"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"gitlab.com/greyxor/slogor"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ racesv1connect.RacePlanServiceHandler = (*Service)(nil)

type RacePlanRepository interface {
	GetRace(ctx context.Context, userID, raceID pgtype.UUID) (*races.Race, error)
	ListRaces(ctx context.Context, userID pgtype.UUID) ([]*races.Race, error)
	CreateRacePlan(ctx context.Context, userID, raceID pgtype.UUID, sections []byte) (*races.RacePlan, error)
	GetRacePlan(ctx context.Context, userID, planID pgtype.UUID) (*races.RacePlan, error)
	GetSharedRacePlan(ctx context.Context, token string) (*races.RacePlan, *races.Race, error)
	ListRacePlans(ctx context.Context, userID pgtype.UUID) ([]*races.RacePlan, error)
	UpdateRacePlanSections(ctx context.Context, userID, planID pgtype.UUID, sections []byte) (*races.RacePlan, error)
	ShareRacePlan(ctx context.Context, userID, planID pgtype.UUID) (*races.RacePlan, error)
	SetRaceDay(ctx context.Context, planID pgtype.UUID, raceDay []byte) (*races.RacePlan, error)
	UnshareRacePlan(ctx context.Context, userID, planID pgtype.UUID) (*races.RacePlan, error)
	DeleteRacePlan(ctx context.Context, userID, planID pgtype.UUID) error
}

type RouteRepository interface {
	GetRoute(ctx context.Context, userID, routeID pgtype.UUID) (*routes.Route, error)
}

// Service implements API handlers for the race plan service.
type Service struct {
	log         *slog.Logger
	repo        RacePlanRepository
	routes      RouteRepository
	liveTracker LiveTracker
	baseURL     string
	nowFn       func() time.Time
}

// NewService creates a new Service. baseURL is the public URL of the server,
// used to create share links.
func NewService(log *slog.Logger, repo RacePlanRepository, routes RouteRepository, liveTracker LiveTracker, baseURL string) *Service {
	return &Service{
		log:         log,
		repo:        repo,
		routes:      routes,
		liveTracker: liveTracker,
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		nowFn:       time.Now,
	}
}

// SharePath is the path of the shared race plan page.
const SharePath = "/share/race-plans/"

// maxSharedCoursePoints is the number of course points in shared plans.
const maxSharedCoursePoints = 1000

func (s *Service) CreateRacePlan(ctx context.Context, req *connect.Request[racesv1.CreateRacePlanRequest]) (*connect.Response[racesv1.CreateRacePlanResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	raceID, err := store.StringToUUID(req.Msg.GetRaceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid race ID: %w", err))
	}
	r, err := s.repo.GetRace(ctx, userID, raceID)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to get race")
	}
	sections, err := encodeSections(seedPlan(r))
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to encode plan")
	}
	stored, err := s.repo.CreateRacePlan(ctx, userID, raceID, sections)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to create race plan")
	}
	plan, analysis, err := s.planWithAnalysis(ctx, stored, r)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(racesv1.CreateRacePlanResponse_builder{
		Plan:     plan,
		Race:     race.RaceToProto(r),
		Analysis: analysis,
	}.Build()), nil
}

func (s *Service) GetRacePlan(ctx context.Context, req *connect.Request[racesv1.GetRacePlanRequest]) (*connect.Response[racesv1.GetRacePlanResponse], error) {
	stored, r, err := s.getPlan(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	plan, analysis, err := s.planWithAnalysis(ctx, stored, r)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(racesv1.GetRacePlanResponse_builder{
		Plan:     plan,
		Race:     race.RaceToProto(r),
		Analysis: analysis,
	}.Build()), nil
}

func (s *Service) ListRacePlans(ctx context.Context, req *connect.Request[racesv1.ListRacePlansRequest]) (*connect.Response[racesv1.ListRacePlansResponse], error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	plans, err := s.repo.ListRacePlans(ctx, userID)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to list race plans")
	}
	raceList, err := s.repo.ListRaces(ctx, userID)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to list races")
	}
	raceByID := map[pgtype.UUID]*races.Race{}
	for _, r := range raceList {
		raceByID[r.ID] = r
	}
	var summaries []*racesv1.RacePlanSummary
	for _, stored := range plans {
		r, ok := raceByID[stored.RaceID]
		if !ok {
			continue
		}
		plan, err := decodeSections(stored.Sections)
		if err != nil {
			return nil, s.toConnectError(ctx, err, "failed to decode race plan")
		}
		// Listing skips the course, which only affects segment elevations.
		analysis := Analyze(plan, raceInfo(r, nil))
		summaries = append(summaries, racesv1.RacePlanSummary_builder{
			Id:          store.UUIDToString(stored.ID),
			Race:        race.RaceToProto(r),
			RulesPassed: analysis.GetRulesPassed(),
			RulesTotal:  analysis.GetRulesTotal(),
			Shared:      stored.ShareToken != "",
			UpdateTime:  timestamppb.New(stored.UpdateTime),
		}.Build())
	}
	sort.SliceStable(summaries, func(i, j int) bool {
		return summaries[i].GetRace().GetDate() < summaries[j].GetRace().GetDate()
	})
	return connect.NewResponse(racesv1.ListRacePlansResponse_builder{Plans: summaries}.Build()), nil
}

// sections are the top level fields of a race plan that can be updated.
var sections = []string{"goals", "checkpoints", "pacing", "nutrition", "gear", "crew", "contingencies", "race_morning", "mental", "notes"}

func (s *Service) UpdateRacePlan(ctx context.Context, req *connect.Request[racesv1.UpdateRacePlanRequest]) (*connect.Response[racesv1.UpdateRacePlanResponse], error) {
	stored, r, err := s.getPlan(ctx, req.Msg.GetPlan().GetId())
	if err != nil {
		return nil, err
	}
	current, err := decodeSections(stored.Sections)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to decode race plan")
	}
	if err := protomask.Apply(current, req.Msg.GetPlan(), req.Msg.GetUpdateMask(), "id", "race_id", "share_url", "create_time", "update_time", "race_day"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%w; valid sections are %s", err, strings.Join(sections, ", ")))
	}
	if err := validatePlan(current); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	encoded, err := encodeSections(current)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to encode race plan")
	}
	updated, err := s.repo.UpdateRacePlanSections(ctx, stored.UserID, stored.ID, encoded)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to update race plan")
	}
	plan, analysis, err := s.planWithAnalysis(ctx, updated, r)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(racesv1.UpdateRacePlanResponse_builder{
		Plan:     plan,
		Race:     race.RaceToProto(r),
		Analysis: analysis,
	}.Build()), nil
}

func (s *Service) DeleteRacePlan(ctx context.Context, req *connect.Request[racesv1.DeleteRacePlanRequest]) (*connect.Response[racesv1.DeleteRacePlanResponse], error) {
	stored, _, err := s.getPlan(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.repo.DeleteRacePlan(ctx, stored.UserID, stored.ID); err != nil {
		return nil, s.toConnectError(ctx, err, "failed to delete race plan")
	}
	return connect.NewResponse(racesv1.DeleteRacePlanResponse_builder{}.Build()), nil
}

func (s *Service) ShareRacePlan(ctx context.Context, req *connect.Request[racesv1.ShareRacePlanRequest]) (*connect.Response[racesv1.ShareRacePlanResponse], error) {
	stored, _, err := s.getPlan(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	shared, err := s.repo.ShareRacePlan(ctx, stored.UserID, stored.ID)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to share race plan")
	}
	return connect.NewResponse(racesv1.ShareRacePlanResponse_builder{
		ShareUrl: s.shareURL(shared.ShareToken),
	}.Build()), nil
}

func (s *Service) UnshareRacePlan(ctx context.Context, req *connect.Request[racesv1.UnshareRacePlanRequest]) (*connect.Response[racesv1.UnshareRacePlanResponse], error) {
	stored, _, err := s.getPlan(ctx, req.Msg.GetId())
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.UnshareRacePlan(ctx, stored.UserID, stored.ID); err != nil {
		return nil, s.toConnectError(ctx, err, "failed to unshare race plan")
	}
	return connect.NewResponse(racesv1.UnshareRacePlanResponse_builder{}.Build()), nil
}

func (s *Service) GetSharedRacePlan(ctx context.Context, req *connect.Request[racesv1.GetSharedRacePlanRequest]) (*connect.Response[racesv1.GetSharedRacePlanResponse], error) {
	// Shared plans don't require authentication.
	stored, r, err := s.repo.GetSharedRacePlan(ctx, req.Msg.GetToken())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("this race plan doesn't exist or is no longer shared"))
		}
		return nil, s.toConnectError(ctx, err, "failed to get shared race plan")
	}
	course, err := s.course(ctx, r)
	if err != nil {
		return nil, err
	}
	plan, err := decodeSections(stored.Sections)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to decode race plan")
	}
	setOutputFields(plan, stored, s.shareURL(stored.ShareToken))
	var coursePoints []*routesv1.RoutePoint
	if len(course) > 0 {
		coursePoints = route.PathToProto(course, maxSharedCoursePoints)
	}
	raceProto := race.RaceToProto(r)
	// The course route belongs to the owner and can't be viewed.
	raceProto.SetCourseRouteId("")
	return connect.NewResponse(racesv1.GetSharedRacePlanResponse_builder{
		Plan:     plan,
		Race:     raceProto,
		Analysis: Analyze(plan, raceInfo(r, course)),
		Course:   coursePoints,
	}.Build()), nil
}

// getPlan gets one of the authenticated user's race plans and its race.
func (s *Service) getPlan(ctx context.Context, id string) (*races.RacePlan, *races.Race, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}
	planID, err := store.StringToUUID(id)
	if err != nil {
		return nil, nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid race plan ID: %w", err))
	}
	stored, err := s.repo.GetRacePlan(ctx, userID, planID)
	if err != nil {
		return nil, nil, s.toConnectError(ctx, err, "failed to get race plan")
	}
	r, err := s.repo.GetRace(ctx, userID, stored.RaceID)
	if err != nil {
		return nil, nil, s.toConnectError(ctx, err, "failed to get race")
	}
	return stored, r, nil
}

// planWithAnalysis decodes a stored plan and analyses it.
func (s *Service) planWithAnalysis(ctx context.Context, stored *races.RacePlan, r *races.Race) (*racesv1.RacePlan, *racesv1.RacePlanAnalysis, error) {
	plan, err := decodeSections(stored.Sections)
	if err != nil {
		return nil, nil, s.toConnectError(ctx, err, "failed to decode race plan")
	}
	setOutputFields(plan, stored, s.shareURL(stored.ShareToken))
	course, err := s.course(ctx, r)
	if err != nil {
		return nil, nil, err
	}
	return plan, Analyze(plan, raceInfo(r, course)), nil
}

// course returns the race's course, or nil if it has none.
func (s *Service) course(ctx context.Context, r *races.Race) ([]geo.Point, error) {
	if !r.CourseRouteID.Valid {
		return nil, nil
	}
	courseRoute, err := s.routes.GetRoute(ctx, r.UserID, r.CourseRouteID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, s.toConnectError(ctx, err, "failed to get course route")
	}
	points, _ := courseRoute.Path()
	return points, nil
}

func (s *Service) shareURL(token string) string {
	if token == "" {
		return ""
	}
	return s.baseURL + SharePath + token
}

func raceInfo(r *races.Race, course []geo.Point) RaceInfo {
	return RaceInfo{
		DistanceMeters: r.DistanceMeters,
		StartTime:      r.StartTime,
		CutoffSeconds:  r.CutoffSeconds,
		Course:         course,
	}
}

// setOutputFields sets the fields of a plan that are not stored in its
// sections.
func setOutputFields(plan *racesv1.RacePlan, stored *races.RacePlan, shareURL string) {
	if raceDay, err := decodeRaceDay(stored.RaceDay); err == nil {
		plan.SetRaceDay(raceDay)
	}
	plan.SetId(store.UUIDToString(stored.ID))
	plan.SetRaceId(store.UUIDToString(stored.RaceID))
	plan.SetShareUrl(shareURL)
	plan.SetCreateTime(timestamppb.New(stored.CreateTime))
	plan.SetUpdateTime(timestamppb.New(stored.UpdateTime))
}

// encodeSections encodes the sections of a plan for storage.
func encodeSections(plan *racesv1.RacePlan) ([]byte, error) {
	clone := racesv1.RacePlan_builder{
		Goals:         plan.GetGoals(),
		Checkpoints:   plan.GetCheckpoints(),
		Pacing:        plan.GetPacing(),
		Nutrition:     plan.GetNutrition(),
		Gear:          plan.GetGear(),
		Crew:          plan.GetCrew(),
		Contingencies: plan.GetContingencies(),
		RaceMorning:   plan.GetRaceMorning(),
		Mental:        plan.GetMental(),
		Notes:         plan.GetNotes(),
	}.Build()
	return protojson.Marshal(clone)
}

func decodeSections(data []byte) (*racesv1.RacePlan, error) {
	var plan racesv1.RacePlan
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// Limits on plan sizes, to keep plans reasonable.
const (
	maxCheckpoints = 100
	maxListItems   = 200
)

func validatePlan(plan *racesv1.RacePlan) error {
	if len(plan.GetCheckpoints()) > maxCheckpoints {
		return fmt.Errorf("at most %d checkpoints are allowed", maxCheckpoints)
	}
	for _, n := range []int{len(plan.GetGear()), len(plan.GetContingencies()), len(plan.GetRaceMorning()), len(plan.GetNutrition().GetProducts())} {
		if n > maxListItems {
			return fmt.Errorf("at most %d items are allowed per list", maxListItems)
		}
	}
	for i, cp := range plan.GetCheckpoints() {
		if cp.GetDistanceMeters() < 0 || cp.GetPlannedSegmentSeconds() < 0 || cp.GetPlannedStopSeconds() < 0 || cp.GetCutoffSeconds() < 0 {
			return fmt.Errorf("checkpoint %d: distances and times must not be negative", i+1)
		}
		for _, intake := range cp.GetIntake() {
			if intake.GetQuantity() < 0 {
				return fmt.Errorf("checkpoint %d: intake quantities must not be negative", i+1)
			}
		}
	}
	return nil
}

// seedPlan returns a new plan for a race, with the finish and an empty entry
// for each core contingency scenario.
func seedPlan(r *races.Race) *racesv1.RacePlan {
	var contingencies []*racesv1.Contingency
	for _, scenario := range coreContingencies {
		contingencies = append(contingencies, racesv1.Contingency_builder{Scenario: scenario}.Build())
	}
	return racesv1.RacePlan_builder{
		Checkpoints: []*racesv1.Checkpoint{
			racesv1.Checkpoint_builder{
				Name:           "Finish",
				DistanceMeters: r.DistanceMeters,
				CutoffSeconds:  r.CutoffSeconds,
			}.Build(),
		},
		Contingencies: contingencies,
	}.Build()
}

func (s *Service) toConnectError(ctx context.Context, err error, msg string) error {
	var connectErr *connect.Error
	switch {
	case errors.As(err, &connectErr):
		return err
	case errors.Is(err, store.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New("not found"))
	default:
		s.log.ErrorContext(ctx, msg, slogor.Err(err))
		return connect.NewError(connect.CodeInternal, fmt.Errorf("%s: %w", msg, err))
	}
}
