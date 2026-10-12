// Package mapdata implements the MapDataService, which manages the
// OpenStreetMap data in the routing graph.
package mapdata

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"

	"connectrpc.com/connect/v2"
	"github.com/catzkorn/trail-tools/internal/authn"
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	routingv1 "github.com/catzkorn/trail-tools/internal/gen/routing/v1"
	"github.com/catzkorn/trail-tools/internal/gen/routing/v1/routingv1connect"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/routing"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
	"gitlab.com/greyxor/slogor"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _ routingv1connect.MapDataServiceHandler = (*Service)(nil)

const (
	// maxAreaDegrees limits the size of an imported area, as the graph of a
	// large area doesn't fit in a small database and takes long to import.
	maxAreaDegrees = 3
	// recentImports is the number of imports listed.
	recentImports = 20
)

// regionName is the format of region names.
var regionName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,59}$`)

// ExtractFinder finds the OpenStreetMap extracts covering an area.
type ExtractFinder interface {
	Find(ctx context.Context, bounds geo.Bounds) ([]routing.Extract, error)
}

// ImportRunner runs a queued import, in the background or as a job.
type ImportRunner interface {
	Run(ctx context.Context, id pgtype.UUID) error
}

// Service implements API handlers for the map data service.
type Service struct {
	log      *slog.Logger
	db       *store.DB
	extracts ExtractFinder
	runner   ImportRunner
}

// NewService creates a new Service.
func NewService(log *slog.Logger, db *store.DB, extracts ExtractFinder, runner ImportRunner) *Service {
	return &Service{log: log, db: db, extracts: extracts, runner: runner}
}

func (s *Service) GetMapData(ctx context.Context, req *routingv1.GetMapDataRequest) (*routingv1.GetMapDataResponse, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	regions, err := routing.ListRegions(ctx, s.db)
	if err != nil {
		return nil, s.internalError(ctx, err)
	}
	imports, err := routing.ListImports(ctx, s.db, recentImports)
	if err != nil {
		return nil, s.internalError(ctx, err)
	}
	size, err := routing.DatabaseSize(ctx, s.db)
	if err != nil {
		return nil, s.internalError(ctx, err)
	}
	resp := routingv1.GetMapDataResponse_builder{DatabaseSizeBytes: size}.Build()
	protoRegions := make([]*routingv1.Region, len(regions))
	for i, r := range regions {
		protoRegions[i] = routingv1.Region_builder{
			Name:       r.Name,
			Area:       areaToProto(r.Bounds),
			EdgeCount:  int32(r.EdgeCount),
			SizeBytes:  r.SizeBytes,
			ImportTime: timestamppb.New(r.ImportTime),
		}.Build()
	}
	resp.SetRegions(protoRegions)
	protoImports := make([]*routingv1.Import, len(imports))
	for i, r := range imports {
		protoImports[i] = importToProto(r)
	}
	resp.SetImports(protoImports)
	return resp, nil
}

func (s *Service) FindExtracts(ctx context.Context, req *routingv1.FindExtractsRequest) (*routingv1.FindExtractsResponse, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	bounds, err := areaFromProto(req.GetArea())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}
	extracts, err := s.findExtracts(ctx, bounds)
	if err != nil {
		return nil, err
	}
	return routingv1.FindExtractsResponse_builder{ExtractNames: extractNames(extracts)}.Build(), nil
}

func (s *Service) StartImport(ctx context.Context, req *routingv1.StartImportRequest) (*routingv1.StartImportResponse, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	region := req.GetRegion()
	if !regionName.MatchString(region) {
		return nil, connect.NewError(connect.CodeInvalidArgument, "region must be lowercase letters, digits and dashes, e.g. central-switzerland")
	}
	bounds, err := areaFromProto(req.GetArea())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}
	// Check before looking up the extracts, though only creating the import
	// ensures no other import started meanwhile.
	active, err := routing.HasActiveImport(ctx, s.db)
	if err != nil {
		return nil, s.internalError(ctx, err)
	}
	if active {
		return nil, activeImportError()
	}
	extracts, err := s.findExtracts(ctx, bounds)
	if err != nil {
		return nil, err
	}
	userID, _ := authn.GetUserID(ctx)
	record, err := routing.CreateImport(ctx, s.db, region, bounds, extracts, userID)
	if err != nil {
		if errors.Is(err, routing.ErrActiveImport) {
			return nil, activeImportError()
		}
		return nil, s.internalError(ctx, err)
	}
	if err := s.runner.Run(ctx, record.ID); err != nil {
		s.log.ErrorContext(ctx, "Failed to start an import", slogor.Err(err))
		if err := routing.FailImport(context.WithoutCancel(ctx), s.db, record.ID, "failed to start the import"); err != nil {
			s.log.ErrorContext(ctx, "Failed to record the failed import", slogor.Err(err))
		}
		return nil, connect.NewError(connect.CodeUnavailable, "failed to start the import, try again later")
	}
	return routingv1.StartImportResponse_builder{Import: importToProto(record)}.Build(), nil
}

func (s *Service) DeleteRegion(ctx context.Context, req *routingv1.DeleteRegionRequest) (*routingv1.DeleteRegionResponse, error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := routing.DeleteRegion(ctx, s.db, req.GetName()); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, "region not found")
		}
		return nil, s.internalError(ctx, err)
	}
	return routingv1.DeleteRegionResponse_builder{}.Build(), nil
}

func (s *Service) findExtracts(ctx context.Context, bounds geo.Bounds) ([]routing.Extract, error) {
	extracts, err := s.extracts.Find(ctx, bounds)
	if err != nil {
		if errors.Is(err, routing.ErrNoExtract) || errors.Is(err, routing.ErrTooManyExtracts) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err.Error()).WithCause(err)
		}
		s.log.ErrorContext(ctx, "Failed to find OpenStreetMap extracts", slogor.Err(err))
		return nil, connect.NewError(connect.CodeUnavailable, "failed to look up OpenStreetMap extracts, try again later")
	}
	return extracts, nil
}

// activeImportError is returned when starting an import while another is
// queued or running.
func activeImportError() error {
	return connect.NewError(connect.CodeFailedPrecondition, "an import is already running, wait for it to finish")
}

func extractNames(extracts []routing.Extract) []string {
	names := make([]string, len(extracts))
	for i, x := range extracts {
		names[i] = x.Name
	}
	return names
}

func (s *Service) internalError(ctx context.Context, err error) error {
	s.log.ErrorContext(ctx, "Map data request failed", slogor.Err(err))
	return connect.NewError(connect.CodeInternal, "internal error")
}

// requireAdmin returns an error unless the user is an administrator.
func requireAdmin(ctx context.Context) error {
	user, ok := authn.GetUser(ctx)
	if !ok {
		return connect.NewError(connect.CodeUnauthenticated, "")
	}
	if !user.IsAdmin() {
		return connect.NewError(connect.CodePermissionDenied, "only administrators can manage map data")
	}
	return nil
}

func areaFromProto(area *routingv1.Area) (geo.Bounds, error) {
	b := geo.Bounds{
		MinLat: area.GetSouthWest().GetLat(),
		MinLng: area.GetSouthWest().GetLng(),
		MaxLat: area.GetNorthEast().GetLat(),
		MaxLng: area.GetNorthEast().GetLng(),
	}
	switch {
	case b.MinLat >= b.MaxLat || b.MinLng >= b.MaxLng:
		return geo.Bounds{}, errors.New("the area's south west corner must be south and west of its north east corner")
	case b.MinLat < -90 || b.MaxLat > 90 || b.MinLng < -180 || b.MaxLng > 180:
		return geo.Bounds{}, errors.New("the area is outside the map")
	case b.MaxLat-b.MinLat > maxAreaDegrees || b.MaxLng-b.MinLng > maxAreaDegrees:
		return geo.Bounds{}, fmt.Errorf("the area must be at most %d degrees across", maxAreaDegrees)
	}
	return b, nil
}

func areaToProto(b geo.Bounds) *routingv1.Area {
	return routingv1.Area_builder{
		SouthWest: geov1.LatLng_builder{Lat: b.MinLat, Lng: b.MinLng}.Build(),
		NorthEast: geov1.LatLng_builder{Lat: b.MaxLat, Lng: b.MaxLng}.Build(),
	}.Build()
}

func importToProto(r *routing.ImportRecord) *routingv1.Import {
	b := routingv1.Import_builder{
		Id:           store.UUIDToString(r.ID),
		Region:       r.Region,
		Area:         areaToProto(r.Bounds),
		ExtractNames: extractNames(r.Extracts),
		State:        stateToProto(r.State),
		Message:      r.Message,
		EdgeCount:    int32(r.EdgeCount),
		CreateTime:   timestamppb.New(r.CreateTime),
	}
	if r.FinishTime.Valid {
		b.FinishTime = timestamppb.New(r.FinishTime.Time)
	}
	return b.Build()
}

func stateToProto(state routing.ImportState) routingv1.ImportState {
	switch state {
	case routing.ImportQueued:
		return routingv1.ImportState_IMPORT_STATE_QUEUED
	case routing.ImportRunning:
		return routingv1.ImportState_IMPORT_STATE_RUNNING
	case routing.ImportDone:
		return routingv1.ImportState_IMPORT_STATE_DONE
	case routing.ImportFailed:
		return routingv1.ImportState_IMPORT_STATE_FAILED
	default:
		return routingv1.ImportState_IMPORT_STATE_UNSPECIFIED
	}
}
