package services_test

import (
	"context"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect"
	geov1 "github.com/catzkorn/trail-tools/internal/gen/geo/v1"
	routingv1 "github.com/catzkorn/trail-tools/internal/gen/routing/v1"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/routing"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5/pgtype"
)

// fakeExtracts has one extract, covering everything south of 60 degrees.
type fakeExtracts struct{}

func (fakeExtracts) Find(_ context.Context, bounds geo.Bounds) (routing.Extract, error) {
	if bounds.MaxLat > 60 {
		return routing.Extract{}, routing.ErrNoExtract
	}
	return routing.Extract{ID: "testland", Name: "Testland", URL: "https://example.com/testland.osm.pbf"}, nil
}

// fakeImportRunner records the imports it's asked to run.
type fakeImportRunner struct {
	mu  sync.Mutex
	ids []pgtype.UUID
}

func (f *fakeImportRunner) Run(_ context.Context, id pgtype.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ids = append(f.ids, id)
	return nil
}

func area(south, west, north, east float64) *routingv1.Area {
	return routingv1.Area_builder{
		SouthWest: geov1.LatLng_builder{Lat: south, Lng: west}.Build(),
		NorthEast: geov1.LatLng_builder{Lat: north, Lng: east}.Build(),
	}.Build()
}

func TestMapData(t *testing.T) {
	t.Parallel()
	e := newEnv(t)
	ctx := context.Background()

	// Only administrators can manage map data.
	_, err := e.mapDataAPI.GetMapData(ctx, connect.NewRequest(routingv1.GetMapDataRequest_builder{}.Build()))
	wantCode(t, err, connect.CodePermissionDenied)
	if _, err := e.db.Exec(ctx, "update users set is_admin = true where id = $1", e.userID); err != nil {
		t.Fatal(err)
	}

	found, err := e.mapDataAPI.FindExtract(ctx, connect.NewRequest(routingv1.FindExtractRequest_builder{Area: area(46.6, 7.9, 47.5, 9.2)}.Build()))
	if err != nil || found.Msg.GetExtractName() != "Testland" {
		t.Fatalf("FindExtract: %v %v", found, err)
	}
	_, err = e.mapDataAPI.FindExtract(ctx, connect.NewRequest(routingv1.FindExtractRequest_builder{Area: area(61, 7, 62, 8)}.Build()))
	wantCode(t, err, connect.CodeFailedPrecondition)
	_, err = e.mapDataAPI.FindExtract(ctx, connect.NewRequest(routingv1.FindExtractRequest_builder{Area: area(40, 0, 50, 10)}.Build()))
	wantCode(t, err, connect.CodeInvalidArgument)
	_, err = e.mapDataAPI.StartImport(ctx, connect.NewRequest(routingv1.StartImportRequest_builder{Region: "Central Switzerland", Area: area(46.6, 7.9, 47.5, 9.2)}.Build()))
	wantCode(t, err, connect.CodeInvalidArgument)

	// An import is queued and handed to the runner, and only one runs at a
	// time.
	started, err := e.mapDataAPI.StartImport(ctx, connect.NewRequest(routingv1.StartImportRequest_builder{Region: "central-switzerland", Area: area(46.6, 7.9, 47.5, 9.2)}.Build()))
	if err != nil {
		t.Fatalf("StartImport: %v", err)
	}
	imported := started.Msg.GetImport()
	if imported.GetState() != routingv1.ImportState_IMPORT_STATE_QUEUED || imported.GetExtractName() != "Testland" {
		t.Errorf("unexpected import %v", imported)
	}
	if len(e.imports.ids) != 1 || store.UUIDToString(e.imports.ids[0]) != imported.GetId() {
		t.Errorf("the runner got %v, want %s", e.imports.ids, imported.GetId())
	}
	_, err = e.mapDataAPI.StartImport(ctx, connect.NewRequest(routingv1.StartImportRequest_builder{Region: "other", Area: area(46, 7, 46.5, 7.5)}.Build()))
	wantCode(t, err, connect.CodeFailedPrecondition)

	// Finish the import with a small graph, as running it needs a download.
	bounds := geo.Bounds{MinLat: 46.6, MinLng: 7.9, MaxLat: 47.5, MaxLng: 9.2}
	edges := slices.Values([]routing.Edge{
		{Source: 1, Target: 2, Points: []geo.Point{{Lat: 47, Lng: 8, Ele: 500}, {Lat: 47.01, Lng: 8, Ele: 520}}, Highway: "path", Terrain: geo.TerrainTrail},
		{Source: 2, Target: 3, Points: []geo.Point{{Lat: 47.01, Lng: 8, Ele: 520}, {Lat: 47.02, Lng: 8, Ele: 510}}, Highway: "track", Terrain: geo.TerrainGravel},
	})
	if _, err := routing.SaveRegion(ctx, e.db, "central-switzerland", bounds, edges); err != nil {
		t.Fatal(err)
	}
	id, _ := store.StringToUUID(imported.GetId())
	if err := routing.FailImport(ctx, e.db, id, "the download failed"); err != nil {
		t.Fatal(err)
	}

	data, err := e.mapDataAPI.GetMapData(ctx, connect.NewRequest(routingv1.GetMapDataRequest_builder{}.Build()))
	if err != nil {
		t.Fatalf("GetMapData: %v", err)
	}
	regions := data.Msg.GetRegions()
	if len(regions) != 1 || regions[0].GetName() != "central-switzerland" || regions[0].GetEdgeCount() != 2 || regions[0].GetSizeBytes() <= 0 {
		t.Errorf("unexpected regions %v", regions)
	}
	if regions[0].GetArea().GetNorthEast().GetLat() != 47.5 {
		t.Errorf("unexpected region area %v", regions[0].GetArea())
	}
	imports := data.Msg.GetImports()
	if len(imports) != 1 || imports[0].GetState() != routingv1.ImportState_IMPORT_STATE_FAILED || imports[0].GetMessage() != "the download failed" || !imports[0].HasFinishTime() {
		t.Errorf("unexpected imports %v", imports)
	}
	if data.Msg.GetDatabaseSizeBytes() <= 0 {
		t.Errorf("unexpected database size %d", data.Msg.GetDatabaseSizeBytes())
	}

	// Deleting a region frees its part of the graph.
	if _, err := e.mapDataAPI.DeleteRegion(ctx, connect.NewRequest(routingv1.DeleteRegionRequest_builder{Name: "central-switzerland"}.Build())); err != nil {
		t.Fatalf("DeleteRegion: %v", err)
	}
	_, err = e.mapDataAPI.DeleteRegion(ctx, connect.NewRequest(routingv1.DeleteRegionRequest_builder{Name: "central-switzerland"}.Build()))
	wantCode(t, err, connect.CodeNotFound)
}
