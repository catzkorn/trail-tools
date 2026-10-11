package routing

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/jackc/pgx/v5"
)

// The routing queries use PostGIS and pgRouting functions that sqlc doesn't
// know about, so they are written with pgx directly.

// SaveRegion replaces the edges of a region in the routing graph.
func SaveRegion(ctx context.Context, db *store.DB, region string, bounds geo.Bounds, edges []Edge) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `delete from routing_edges where region = $1`, region); err != nil {
		return fmt.Errorf("failed to delete the region's edges: %w", err)
	}
	// Geometries are copied as text into a staging table, as pgx can't encode
	// PostGIS types.
	if _, err := tx.Exec(ctx, `create temporary table staged_edges (
		osm_way_id bigint, source bigint, target bigint, wkt text, elevations real[],
		length_meters double precision, climb_meters real, descent_meters real,
		highway text, surface text, sac_scale text, terrain text
	) on commit drop`); err != nil {
		return fmt.Errorf("failed to create the staging table: %w", err)
	}
	columns := []string{"osm_way_id", "source", "target", "wkt", "elevations", "length_meters", "climb_meters", "descent_meters", "highway", "surface", "sac_scale", "terrain"}
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"staged_edges"}, columns, pgx.CopyFromSlice(len(edges), func(i int) ([]any, error) {
		e := edges[i]
		elevations := filledElevations(e.Points)
		climb, descent := climbDescent(elevations)
		return []any{
			e.WayID, e.Source, e.Target, lineWKT(e.Points), elevations,
			pathLength(e.Points), climb, descent,
			e.Highway, e.Surface, e.SacScale, string(e.Terrain),
		}, nil
	}))
	if err != nil {
		return fmt.Errorf("failed to copy edges: %w", err)
	}
	if _, err := tx.Exec(ctx, `insert into routing_edges (
		region, osm_way_id, source, target, geom, elevations, length_meters, climb_meters,
		descent_meters, highway, surface, sac_scale, terrain
	) select $1, osm_way_id, source, target, ST_GeomFromText(wkt, 4326), elevations,
		length_meters, climb_meters, descent_meters, highway, surface, sac_scale, terrain
	from staged_edges`, region); err != nil {
		return fmt.Errorf("failed to insert edges: %w", err)
	}
	if _, err := tx.Exec(ctx, `insert into routing_regions (name, bounds, edge_count)
	values ($1, ST_MakeEnvelope($2, $3, $4, $5, 4326), $6)
	on conflict (name) do update set
		bounds = excluded.bounds, edge_count = excluded.edge_count, import_time = current_timestamp`,
		region, bounds.MinLng, bounds.MinLat, bounds.MaxLng, bounds.MaxLat, len(edges)); err != nil {
		return fmt.Errorf("failed to save the region: %w", err)
	}
	return tx.Commit(ctx)
}

// lineWKT returns the well-known text of a line through the points.
func lineWKT(points []geo.Point) string {
	var b strings.Builder
	b.WriteString("LINESTRING(")
	for i, p := range points {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(p.Lng, 'f', 7, 64))
		b.WriteByte(' ')
		b.WriteString(strconv.FormatFloat(p.Lat, 'f', 7, 64))
	}
	b.WriteByte(')')
	return b.String()
}

// filledElevations returns the elevations of the points, filling unknown
// ones from the nearest known elevation before them, or after them at the
// start. They are 0 if none are known.
func filledElevations(points []geo.Point) []float32 {
	elevations := make([]float32, len(points))
	last := math.NaN()
	for i, p := range points {
		if !math.IsNaN(p.Ele) {
			last = p.Ele
		}
		elevations[i] = float32(last)
	}
	first := float32(0)
	for _, e := range elevations {
		if !math.IsNaN(float64(e)) {
			first = e
			break
		}
	}
	for i, e := range elevations {
		if !math.IsNaN(float64(e)) {
			break
		}
		elevations[i] = first
	}
	return elevations
}

// climbDescent returns the total elevation gained and lost along a path.
func climbDescent(elevations []float32) (climb, descent float32) {
	for i := 1; i < len(elevations); i++ {
		if d := elevations[i] - elevations[i-1]; d > 0 {
			climb += d
		} else {
			descent -= d
		}
	}
	return climb, descent
}

func pathLength(points []geo.Point) float64 {
	var length float64
	for i := 1; i < len(points); i++ {
		length += geo.Distance(points[i-1], points[i])
	}
	return length
}
