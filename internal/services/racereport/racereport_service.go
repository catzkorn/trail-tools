// Package racereport implements the RaceReportService.
package racereport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect/v2"
	"github.com/catzkorn/trail-tools/internal/activities"
	"github.com/catzkorn/trail-tools/internal/authn"
	activitiesv1 "github.com/catzkorn/trail-tools/internal/gen/activities/v1"
	racesv1 "github.com/catzkorn/trail-tools/internal/gen/races/v1"
	"github.com/catzkorn/trail-tools/internal/gen/races/v1/racesv1connect"
	"github.com/catzkorn/trail-tools/internal/races"
	"github.com/catzkorn/trail-tools/internal/services/raceplan"
	"github.com/catzkorn/trail-tools/internal/services/trip"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/catzkorn/trail-tools/internal/trips"
	"github.com/jackc/pgx/v5/pgtype"
	"gitlab.com/greyxor/slogor"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ racesv1connect.RaceReportServiceHandler = (*Service)(nil)

// RacePath is the path segment of race reports on public profiles:
// /u/<handle>/races/<slug>.
const RacePath = "races"

const (
	// maxTitleLength is the longest title in characters.
	maxTitleLength = 200
	// maxSectionLength is the longest section of a report in characters.
	maxSectionLength = 50_000
	// maxPlacementLength is the longest placement in characters.
	maxPlacementLength = 200
	// maxSlugAttempts is how many slugs are tried for a new report.
	maxSlugAttempts = 20
	// maxActivitiesPerDay is how many activities on the race date are
	// considered to find the race.
	maxActivitiesPerDay = 50
)

// ReportRepository stores race reports and races.
type ReportRepository interface {
	GetRace(ctx context.Context, userID, raceID pgtype.UUID) (*races.Race, error)
	CreateRaceReport(ctx context.Context, report *races.RaceReport) (*races.RaceReport, error)
	GetRaceReport(ctx context.Context, userID, reportID pgtype.UUID) (*races.RaceReport, error)
	GetRaceReportBySlug(ctx context.Context, userID pgtype.UUID, slug string) (*races.RaceReport, error)
	ListRaceReports(ctx context.Context, userID pgtype.UUID, filter races.RaceReportFilter) ([]*races.RaceReport, error)
	UpdateRaceReport(ctx context.Context, update *races.RaceReportUpdate) (*races.RaceReport, error)
	SetRaceReportPublished(ctx context.Context, userID, reportID pgtype.UUID, published bool) (*races.RaceReport, error)
	DeleteRaceReport(ctx context.Context, userID, reportID pgtype.UUID) error
}

// ActivityRepository gets activities.
type ActivityRepository interface {
	GetActivity(ctx context.Context, userID, activityID pgtype.UUID) (*activities.Activity, error)
	GetActivityStreams(ctx context.Context, userID, activityID pgtype.UUID) ([]byte, error)
	ListActivities(ctx context.Context, userID pgtype.UUID, filter activities.ListActivitiesFilter) ([]*activities.Activity, error)
}

// ProfileRepository gets the public profiles reports are published on.
type ProfileRepository interface {
	GetProfile(ctx context.Context, userID pgtype.UUID) (*trips.Profile, error)
	GetProfileByHandle(ctx context.Context, handle string) (*trips.Profile, error)
}

// Plans returns what a race's plan planned, e.g. a raceplan.Service.
type Plans interface {
	PlanOutcome(ctx context.Context, userID, raceID pgtype.UUID) (*raceplan.PlanOutcome, error)
}

// Service implements API handlers for the race report service.
type Service struct {
	log        *slog.Logger
	repo       ReportRepository
	activities ActivityRepository
	profiles   ProfileRepository
	plans      Plans
	baseURL    string
}

// NewService creates a new Service. baseURL is the public URL of the server,
// used for the URLs of reports.
func NewService(log *slog.Logger, repo ReportRepository, activities ActivityRepository, profiles ProfileRepository, plans Plans, baseURL string) *Service {
	return &Service{
		log:        log,
		repo:       repo,
		activities: activities,
		profiles:   profiles,
		plans:      plans,
		baseURL:    strings.TrimSuffix(baseURL, "/"),
	}
}

func (s *Service) CreateRaceReport(ctx context.Context, req *racesv1.CreateRaceReportRequest) (*racesv1.CreateRaceReportResponse, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	raceID, err := store.StringToUUID(req.GetRaceId())
	if err != nil {
		return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid race ID: %v", err).WithCause(err)
	}
	race, err := s.repo.GetRace(ctx, userID, raceID)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to get race")
	}
	report := &races.RaceReport{
		UserID:              userID,
		RaceID:              race.ID,
		Title:               race.Name,
		RaceName:            race.Name,
		RaceDate:            race.Date,
		DistanceMeters:      race.DistanceMeters,
		ElevationGainMeters: race.ElevationGainMeters,
	}
	activity, err := s.raceActivity(ctx, userID, race, req.GetActivityId())
	if err != nil {
		return nil, err
	}
	if activity != nil {
		report.ActivityID = activity.ID
		report.FinishSeconds = activity.ElapsedTimeSeconds
	}
	base := trip.Slugify(report.Title)
	if base == "" {
		base = "race"
	}
	for i := 1; i <= maxSlugAttempts; i++ {
		report.Slug = base
		if i > 1 {
			report.Slug = fmt.Sprintf("%s-%d", base, i)
		}
		created, err := s.repo.CreateRaceReport(ctx, report)
		if errors.Is(err, races.ErrSlugTaken) {
			continue
		}
		if err != nil {
			return nil, s.toConnectError(ctx, err, "failed to create race report")
		}
		reportProto, comparison, err := s.withComparison(ctx, created)
		if err != nil {
			return nil, err
		}
		return racesv1.CreateRaceReportResponse_builder{Report: reportProto, Comparison: comparison}.Build(), nil
	}
	return nil, connect.NewError(connect.CodeAlreadyExists, "too many race reports have this title")
}

// raceActivity returns the activity with the ID, or if it's empty, the
// user's longest activity on the race date, or nil if there is none.
func (s *Service) raceActivity(ctx context.Context, userID pgtype.UUID, race *races.Race, id string) (*activities.Activity, error) {
	if id != "" {
		activityID, err := store.StringToUUID(id)
		if err != nil {
			return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid activity ID: %v", err).WithCause(err)
		}
		activity, err := s.activities.GetActivity(ctx, userID, activityID)
		if err != nil {
			return nil, s.toConnectError(ctx, err, "failed to get activity")
		}
		return activity, nil
	}
	onDate, err := s.activities.ListActivities(ctx, userID, activities.ListActivitiesFilter{
		StartDate: race.Date,
		EndDate:   race.Date,
		Limit:     maxActivitiesPerDay,
	})
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to list activities")
	}
	var longest *activities.Activity
	for _, a := range onDate {
		if longest == nil || a.DistanceMeters > longest.DistanceMeters {
			longest = a
		}
	}
	return longest, nil
}

func (s *Service) GetRaceReport(ctx context.Context, req *racesv1.GetRaceReportRequest) (*racesv1.GetRaceReportResponse, error) {
	report, err := s.getReport(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	reportProto, comparison, err := s.withComparison(ctx, report)
	if err != nil {
		return nil, err
	}
	return racesv1.GetRaceReportResponse_builder{Report: reportProto, Comparison: comparison}.Build(), nil
}

func (s *Service) ListRaceReports(ctx context.Context, req *racesv1.ListRaceReportsRequest) (*racesv1.ListRaceReportsResponse, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	filter := races.RaceReportFilter{}
	if req.GetRaceId() != "" {
		raceID, err := store.StringToUUID(req.GetRaceId())
		if err != nil {
			return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid race ID: %v", err).WithCause(err)
		}
		filter.RaceID = raceID
	}
	list, err := s.repo.ListRaceReports(ctx, userID, filter)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to list race reports")
	}
	profile, err := s.profile(ctx, userID)
	if err != nil {
		return nil, err
	}
	reports := make([]*racesv1.RaceReport, len(list))
	for i, r := range list {
		reports[i] = s.reportToProto(r, profile)
	}
	return racesv1.ListRaceReportsResponse_builder{Reports: reports}.Build(), nil
}

func (s *Service) UpdateRaceReport(ctx context.Context, req *racesv1.UpdateRaceReportRequest) (*racesv1.UpdateRaceReportResponse, error) {
	current, err := s.getReport(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	update := &races.RaceReportUpdate{ID: current.ID, UserID: current.UserID}
	if req.HasTitle() {
		title := strings.TrimSpace(req.GetTitle())
		switch {
		case title == "":
			return nil, connect.NewError(connect.CodeInvalidArgument, "the title is required")
		case utf8.RuneCountInString(title) > maxTitleLength:
			return nil, connect.Errorf(connect.CodeInvalidArgument, "the title must be at most %d characters", maxTitleLength)
		}
		update.Title = &title
	}
	if req.HasSlug() {
		slug := req.GetSlug()
		if err := trip.ValidateSlug(slug); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		}
		if current.Published && current.Slug != slug {
			return nil, connect.NewError(connect.CodeFailedPrecondition, "the slug of a published report can't be changed, so that links to it keep working; unpublish the report first")
		}
		update.Slug = &slug
	}
	if req.HasActivityId() {
		update.SetActivity = true
		if req.GetActivityId() != "" {
			activityID, err := store.StringToUUID(req.GetActivityId())
			if err != nil {
				return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid activity ID: %v", err).WithCause(err)
			}
			if _, err := s.activities.GetActivity(ctx, current.UserID, activityID); err != nil {
				return nil, s.toConnectError(ctx, err, "failed to get activity")
			}
			update.ActivityID = activityID
		}
	}
	if req.HasFinishSeconds() {
		finish := req.GetFinishSeconds()
		if finish < 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, "the finish time must not be negative")
		}
		update.FinishSeconds = &finish
	}
	if req.HasPlacement() {
		placement := strings.TrimSpace(req.GetPlacement())
		if utf8.RuneCountInString(placement) > maxPlacementLength {
			return nil, connect.Errorf(connect.CodeInvalidArgument, "the placement must be at most %d characters", maxPlacementLength)
		}
		update.Placement = &placement
	}
	for _, section := range []struct {
		name  string
		has   bool
		value string
		set   **string
	}{
		{"summary", req.HasSummary(), req.GetSummary(), &update.Summary},
		{"preparation", req.HasPreparation(), req.GetPreparation(), &update.Preparation},
		{"race day", req.HasRaceDay(), req.GetRaceDay(), &update.RaceDay},
		{"nutrition", req.HasNutrition(), req.GetNutrition(), &update.Nutrition},
		{"what went well", req.HasWentWell(), req.GetWentWell(), &update.WentWell},
		{"what went badly", req.HasWentBadly(), req.GetWentBadly(), &update.WentBadly},
		{"lessons", req.HasLessons(), req.GetLessons(), &update.Lessons},
	} {
		if !section.has {
			continue
		}
		if utf8.RuneCountInString(section.value) > maxSectionLength {
			return nil, connect.Errorf(connect.CodeInvalidArgument, "the %s section must be at most %d characters", section.name, maxSectionLength)
		}
		*section.set = &section.value
	}
	updated, err := s.repo.UpdateRaceReport(ctx, update)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to update race report")
	}
	reportProto, comparison, err := s.withComparison(ctx, updated)
	if err != nil {
		return nil, err
	}
	return racesv1.UpdateRaceReportResponse_builder{Report: reportProto, Comparison: comparison}.Build(), nil
}

func (s *Service) DeleteRaceReport(ctx context.Context, req *racesv1.DeleteRaceReportRequest) (*racesv1.DeleteRaceReportResponse, error) {
	report, err := s.getReport(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.repo.DeleteRaceReport(ctx, report.UserID, report.ID); err != nil {
		return nil, s.toConnectError(ctx, err, "failed to delete race report")
	}
	return racesv1.DeleteRaceReportResponse_builder{}.Build(), nil
}

func (s *Service) PublishRaceReport(ctx context.Context, req *racesv1.PublishRaceReportRequest) (*racesv1.PublishRaceReportResponse, error) {
	report, err := s.getReport(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	profile, err := s.profile(ctx, report.UserID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, "set up a public profile with a handle in Settings before publishing")
	}
	published, err := s.repo.SetRaceReportPublished(ctx, report.UserID, report.ID, true)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to publish race report")
	}
	return racesv1.PublishRaceReportResponse_builder{Report: s.reportToProto(published, profile)}.Build(), nil
}

func (s *Service) UnpublishRaceReport(ctx context.Context, req *racesv1.UnpublishRaceReportRequest) (*racesv1.UnpublishRaceReportResponse, error) {
	report, err := s.getReport(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	unpublished, err := s.repo.SetRaceReportPublished(ctx, report.UserID, report.ID, false)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to unpublish race report")
	}
	profile, err := s.profile(ctx, report.UserID)
	if err != nil {
		return nil, err
	}
	return racesv1.UnpublishRaceReportResponse_builder{Report: s.reportToProto(unpublished, profile)}.Build(), nil
}

func (s *Service) ListPublicRaceReports(ctx context.Context, req *racesv1.ListPublicRaceReportsRequest) (*racesv1.ListPublicRaceReportsResponse, error) {
	// Published reports don't require authentication.
	profile, err := s.profiles.GetProfileByHandle(ctx, req.GetHandle())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "this profile doesn't exist")
		}
		return nil, s.toConnectError(ctx, err, "failed to get profile")
	}
	list, err := s.repo.ListRaceReports(ctx, profile.UserID, races.RaceReportFilter{PublishedOnly: true})
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to list race reports")
	}
	reports := make([]*racesv1.RaceReport, len(list))
	for i, r := range list {
		reports[i] = s.reportToProto(r, profile)
	}
	return racesv1.ListPublicRaceReportsResponse_builder{Reports: reports}.Build(), nil
}

func (s *Service) GetPublicRaceReport(ctx context.Context, req *racesv1.GetPublicRaceReportRequest) (*racesv1.GetPublicRaceReportResponse, error) {
	// Published reports don't require authentication.
	profile, report, err := s.PublicReport(ctx, req.GetHandle(), req.GetSlug())
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "this race report doesn't exist or isn't published")
		}
		return nil, s.toConnectError(ctx, err, "failed to get race report")
	}
	reportProto, comparison, err := s.withComparison(ctx, report)
	if err != nil {
		return nil, err
	}
	return racesv1.GetPublicRaceReportResponse_builder{
		Report:     reportProto,
		Comparison: comparison,
		AuthorName: profile.DisplayName,
	}.Build(), nil
}

// PublicReport gets a race report by its author's handle and slug, if it is
// published or the caller is its author. Otherwise it returns
// store.ErrNotFound, so drafts can't be told apart from missing reports.
func (s *Service) PublicReport(ctx context.Context, handle, slug string) (*trips.Profile, *races.RaceReport, error) {
	profile, err := s.profiles.GetProfileByHandle(ctx, handle)
	if err != nil {
		return nil, nil, err
	}
	report, err := s.repo.GetRaceReportBySlug(ctx, profile.UserID, slug)
	if err != nil {
		return nil, nil, err
	}
	if !report.Published {
		if userID, ok := authn.GetUserID(ctx); !ok || userID != report.UserID {
			return nil, nil, store.ErrNotFound
		}
	}
	return profile, report, nil
}

// withComparison returns a report and its comparison with the race plan.
func (s *Service) withComparison(ctx context.Context, report *races.RaceReport) (*racesv1.RaceReport, *racesv1.RaceComparison, error) {
	profile, err := s.profile(ctx, report.UserID)
	if err != nil {
		return nil, nil, err
	}
	var outcome *raceplan.PlanOutcome
	if report.RaceID.Valid {
		if outcome, err = s.plans.PlanOutcome(ctx, report.UserID, report.RaceID); err != nil && !errors.Is(err, store.ErrNotFound) {
			return nil, nil, s.toConnectError(ctx, err, "failed to get the race plan")
		}
	}
	var rec *recording
	if report.ActivityID.Valid {
		data, err := s.activities.GetActivityStreams(ctx, report.UserID, report.ActivityID)
		switch {
		case err == nil:
			streams := &activitiesv1.ActivityStreams{}
			if err := proto.Unmarshal(data, streams); err != nil {
				return nil, nil, s.toConnectError(ctx, err, "failed to decode activity streams")
			}
			rec = newRecording(streams)
		case !errors.Is(err, store.ErrNotFound):
			return nil, nil, s.toConnectError(ctx, err, "failed to get activity streams")
		}
	}
	comparison := compare(outcome, report.FinishSeconds, report.DistanceMeters, rec)
	return s.reportToProto(report, profile), comparison, nil
}

// getReport gets one of the authenticated user's race reports by ID.
func (s *Service) getReport(ctx context.Context, id string) (*races.RaceReport, error) {
	userID, ok := authn.GetUserID(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, "")
	}
	reportID, err := store.StringToUUID(id)
	if err != nil {
		return nil, connect.Errorf(connect.CodeInvalidArgument, "invalid race report ID: %v", err).WithCause(err)
	}
	report, err := s.repo.GetRaceReport(ctx, userID, reportID)
	if err != nil {
		return nil, s.toConnectError(ctx, err, "failed to get race report")
	}
	return report, nil
}

// profile returns the user's public profile, or nil if the user has none.
func (s *Service) profile(ctx context.Context, userID pgtype.UUID) (*trips.Profile, error) {
	profile, err := s.profiles.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, nil
		}
		return nil, s.toConnectError(ctx, err, "failed to get profile")
	}
	return profile, nil
}

// ReportURL returns the public URL of a race report.
func (s *Service) ReportURL(handle, slug string) string {
	return s.baseURL + trip.ProfilePath + handle + "/" + RacePath + "/" + slug
}

func (s *Service) reportToProto(r *races.RaceReport, profile *trips.Profile) *racesv1.RaceReport {
	b := racesv1.RaceReport_builder{
		Id:                  store.UUIDToString(r.ID),
		RaceId:              store.UUIDToString(r.RaceID),
		ActivityId:          store.UUIDToString(r.ActivityID),
		Title:               r.Title,
		Slug:                r.Slug,
		RaceName:            r.RaceName,
		RaceDate:            store.FormatDate(r.RaceDate),
		DistanceMeters:      r.DistanceMeters,
		ElevationGainMeters: r.ElevationGainMeters,
		FinishSeconds:       r.FinishSeconds,
		Placement:           r.Placement,
		Summary:             r.Sections.Summary,
		Preparation:         r.Sections.Preparation,
		RaceDay:             r.Sections.RaceDay,
		Nutrition:           r.Sections.Nutrition,
		WentWell:            r.Sections.WentWell,
		WentBadly:           r.Sections.WentBadly,
		Lessons:             r.Sections.Lessons,
		Published:           r.Published,
		CreateTime:          timestamppb.New(r.CreateTime),
		UpdateTime:          timestamppb.New(r.UpdateTime),
	}
	if !r.PublishTime.IsZero() {
		b.PublishTime = timestamppb.New(r.PublishTime)
	}
	if profile != nil {
		b.Url = s.ReportURL(profile.Handle, r.Slug)
	}
	return b.Build()
}

func (s *Service) toConnectError(ctx context.Context, err error, msg string) error {
	var connectErr *connect.Error
	switch {
	case errors.As(err, &connectErr):
		return err
	case errors.Is(err, store.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, "not found")
	case errors.Is(err, races.ErrSlugTaken):
		return connect.NewError(connect.CodeAlreadyExists, "another of your race reports has this slug; choose another one")
	default:
		s.log.ErrorContext(ctx, msg, slogor.Err(err))
		return connect.NewError(connect.CodeInternal, msg).WithCause(err)
	}
}
