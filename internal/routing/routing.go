// Package routing routes paths for running on OpenStreetMap data with
// pgRouting, and imports the data into the routing graph.
package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/johanbrandhorst/trail-tools/internal/geo"
	"github.com/johanbrandhorst/trail-tools/internal/store"
)

// Profile controls which paths are preferred.
type Profile int

const (
	// ProfileTrail prefers trails and unpaved paths and avoids busy roads and
	// climbing.
	ProfileTrail Profile = iota
	// ProfileShortest takes the shortest path.
	ProfileShortest
)

// ErrNotRoutable is wrapped by errors for points that can't be routed
// between, such as points far from any mapped path.
var ErrNotRoutable = errors.New("not routable")

const (
	// maxSnapMeters is the furthest a point can be from a mapped path.
	maxSnapMeters = 500
	// minSearchMarginMeters is the smallest margin around the points within
	// which paths are searched.
	minSearchMarginMeters = 2000
	// searchAttempts is the number of times the search area is grown when no
	// path is found.
	searchAttempts = 2
	// preferMeters is how close an edge must be to a preferred path to count
	// as on it.
	preferMeters = 25
	// preferCostFactor scales the cost of edges on preferred paths.
	preferCostFactor = 0.5
	// metersPerDegreeLat is the approximate length of a degree of latitude.
	metersPerDegreeLat = 111320.0
)

// trailCost is the cost of traveling an edge forward with the trail profile:
// its length, scaled by how pleasant it is to run on, plus the effort of
// climbing, a climbed meter costing like 8 meters on the flat.
const trailCost = `length_meters * (case highway
	when 'path' then 1.0 when 'footway' then 1.0 when 'bridleway' then 1.05 when 'track' then 1.05
	when 'steps' then 1.3 when 'pedestrian' then 1.2 when 'living_street' then 1.3
	when 'cycleway' then 1.4 when 'service' then 1.4 when 'unclassified' then 1.4
	when 'residential' then 1.5 when 'road' then 1.5
	when 'tertiary' then 2.0 when 'tertiary_link' then 2.0
	when 'secondary' then 3.0 when 'secondary_link' then 3.0
	when 'primary' then 5.0 when 'primary_link' then 5.0
	else 1.5 end)
* (case when terrain = 'paved' and highway in ('path', 'footway', 'track', 'bridleway') then 1.2 else 1.0 end)
* (case sac_scale when 'demanding_mountain_hiking' then 1.3 when 'alpine_hiking' then 1.8
	when 'demanding_alpine_hiking' then 3.0 when 'difficult_alpine_hiking' then 5.0 else 1.0 end)`

// costs returns the SQL expressions for the cost of traveling edges forward
// and backward with a profile.
func costs(profile Profile, prefer bool) (cost, reverseCost string) {
	if profile == ProfileShortest {
		cost, reverseCost = "length_meters", "length_meters"
	} else {
		cost = "(" + trailCost + " + 8 * climb_meters + 2 * descent_meters)"
		reverseCost = "(" + trailCost + " + 8 * descent_meters + 2 * climb_meters)"
	}
	if prefer {
		// An edge is on a preferred path if its middle and quarter points are
		// near one.
		factor := fmt.Sprintf(`(case when exists (select 1 from preferred_paths p
			where ST_DWithin(ST_LineInterpolatePoint(routing_edges.geom, 0.25), p.geom, p.tolerance)
			and ST_DWithin(ST_LineInterpolatePoint(routing_edges.geom, 0.5), p.geom, p.tolerance)
			and ST_DWithin(ST_LineInterpolatePoint(routing_edges.geom, 0.75), p.geom, p.tolerance)
		) then %g else 1 end)`, preferCostFactor)
		cost, reverseCost = cost+" * "+factor, reverseCost+" * "+factor
	}
	return cost, reverseCost
}

// Request is a request for a path between two points.
type Request struct {
	From, To geo.Point
	Profile  Profile
	// Paths to prefer, such as the user's previous activities. Edges along
	// them cost less, so routes follow them where they reasonably can.
	Prefer [][]geo.Point
}

// Path is a routed path.
type Path struct {
	// The points of the path, with elevations. It starts and ends where the
	// requested points were snapped onto the map.
	Points []geo.Point
	// The terrain of each segment: Terrains[i] is the terrain between
	// Points[i] and Points[i+1]. It has the same length as Points.
	Terrains []geo.Terrain
}

// Router routes paths on the routing graph.
type Router struct {
	db *store.DB
}

// NewRouter creates a Router.
func NewRouter(db *store.DB) *Router {
	return &Router{db: db}
}

// Regions returns the bounds of the imported regions, the areas paths can be
// routed in.
func (r *Router) Regions(ctx context.Context) ([]geo.Bounds, error) {
	rows, err := r.db.Query(ctx, `select ST_YMin(bounds), ST_XMin(bounds), ST_YMax(bounds), ST_XMax(bounds) from routing_regions`)
	if err != nil {
		return nil, fmt.Errorf("failed to list the imported regions: %w", err)
	}
	regions, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (geo.Bounds, error) {
		var b geo.Bounds
		err := row.Scan(&b.MinLat, &b.MinLng, &b.MaxLat, &b.MaxLng)
		return b, err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list the imported regions: %w", err)
	}
	return regions, nil
}

// edge is an edge of the routing graph with its geometry.
type edge struct {
	id, source, target int64
	points             []geo.Point
	// The distance along the edge to each point, in meters.
	dists   []float64
	terrain geo.Terrain
}

// position is a point on an edge, as the distance along it in meters.
type position struct {
	edge *edge
	dist float64
}

const edgeColumns = `id, source, target, ST_AsGeoJSON(geom), elevations, terrain`

// scanEdge scans the edgeColumns of a row, followed by any extra columns.
func scanEdge(row pgx.Row, extra ...any) (*edge, error) {
	var e edge
	var geojson string
	var elevations []float32
	var terrain string
	dest := append([]any{&e.id, &e.source, &e.target, &geojson, &elevations, &terrain}, extra...)
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	var line struct {
		Coordinates [][2]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal([]byte(geojson), &line); err != nil {
		return nil, fmt.Errorf("failed to decode edge %d: %w", e.id, err)
	}
	for i, c := range line.Coordinates {
		p := geo.Point{Lng: c[0], Lat: c[1]}
		if i < len(elevations) {
			p.Ele = float64(elevations[i])
		}
		e.points = append(e.points, p)
	}
	e.dists = geo.CumulativeDistances(e.points)
	e.terrain = geo.Terrain(terrain)
	return &e, nil
}

// length returns the length of the edge in meters.
func (e *edge) length() float64 {
	return e.dists[len(e.dists)-1]
}

// Route computes a path between two points.
func (r *Router) Route(ctx context.Context, req Request) (*Path, error) {
	from, err := snap(ctx, r.db, req.From)
	if err != nil {
		return nil, err
	}
	to, err := snap(ctx, r.db, req.To)
	if err != nil {
		return nil, err
	}
	// The preferred paths are stored in a temporary table, which the search
	// needs a transaction for. Without them, it runs without one, saving the
	// round trips of starting and ending it.
	var db store.Querier = r.db
	prefer := len(req.Prefer) > 0
	if prefer {
		tx, err := r.db.Begin(ctx)
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)
		if err := savePreferred(ctx, tx, req.Prefer); err != nil {
			return nil, err
		}
		db = tx
	}
	cost, reverseCost := costs(req.Profile, prefer)
	margin := math.Max(minSearchMarginMeters, geo.Distance(req.From, req.To)/2)
	for range searchAttempts {
		path, err := search(ctx, db, from, to, cost, reverseCost, margin)
		if err != nil || path != nil {
			return path, err
		}
		margin *= 4
	}
	return nil, fmt.Errorf("%w: no path was found between the points", ErrNotRoutable)
}

// snap returns the position on the nearest edge to a point.
func snap(ctx context.Context, db store.Querier, p geo.Point) (position, error) {
	row := db.QueryRow(ctx, `select `+edgeColumns+`,
		ST_Distance(geom::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography)
	from routing_edges
	order by geom <-> ST_SetSRID(ST_MakePoint($1, $2), 4326)
	limit 1`, p.Lng, p.Lat)
	var distance float64
	e, err := scanEdge(row, &distance)
	if errors.Is(err, pgx.ErrNoRows) {
		return position{}, fmt.Errorf("%w: no map data has been imported; run import-osm for the area", ErrNotRoutable)
	}
	if err != nil {
		return position{}, fmt.Errorf("failed to find the nearest path: %w", err)
	}
	if distance > maxSnapMeters {
		var covered bool
		if err := db.QueryRow(ctx, `select exists (select 1 from routing_regions
			where ST_Contains(bounds, ST_SetSRID(ST_MakePoint($1, $2), 4326)))`, p.Lng, p.Lat).Scan(&covered); err != nil {
			return position{}, fmt.Errorf("failed to check the imported regions: %w", err)
		}
		if !covered {
			return position{}, fmt.Errorf("%w: no map data has been imported for the area around %.5f, %.5f; run import-osm for it", ErrNotRoutable, p.Lat, p.Lng)
		}
		return position{}, fmt.Errorf("%w: there is no mapped path within %d meters of %.5f, %.5f", ErrNotRoutable, maxSnapMeters, p.Lat, p.Lng)
	}
	return position{edge: e, dist: project(e, p)}, nil
}

// project returns the distance along an edge to the point nearest p.
func project(e *edge, p geo.Point) float64 {
	best, bestDist := math.Inf(1), 0.0
	scale := math.Cos(p.Lat * math.Pi / 180)
	for i := 1; i < len(e.points); i++ {
		a, b := e.points[i-1], e.points[i]
		ax, ay := (a.Lng-p.Lng)*scale, a.Lat-p.Lat
		bx, by := (b.Lng-p.Lng)*scale, b.Lat-p.Lat
		dx, dy := bx-ax, by-ay
		f := 0.0
		if l := dx*dx + dy*dy; l > 0 {
			f = math.Max(0, math.Min(1, -(ax*dx+ay*dy)/l))
		}
		x, y := ax+f*dx, ay+f*dy
		if d := x*x + y*y; d < best {
			best = d
			bestDist = e.dists[i-1] + f*(e.dists[i]-e.dists[i-1])
		}
	}
	return bestDist
}

// savePreferred stores the preferred paths in a temporary table for the
// transaction.
func savePreferred(ctx context.Context, tx pgx.Tx, paths [][]geo.Point) error {
	if _, err := tx.Exec(ctx, `create temporary table preferred_paths (
		geom geometry(LineString, 4326), tolerance double precision
	) on commit drop`); err != nil {
		return fmt.Errorf("failed to create the preferred paths table: %w", err)
	}
	batch := &pgx.Batch{}
	for _, path := range paths {
		if len(path) < 2 {
			continue
		}
		// The tolerance in degrees, measured along a parallel, which is the
		// shorter direction, so it is at least preferMeters in any direction.
		tolerance := preferMeters / (metersPerDegreeLat * math.Max(0.01, math.Cos(path[0].Lat*math.Pi/180)))
		batch.Queue(`insert into preferred_paths values (ST_GeomFromText($1, 4326), $2)`, lineWKT(path), tolerance)
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("failed to save the preferred paths: %w", err)
	}
	if _, err := tx.Exec(ctx, `create index on preferred_paths using gist (geom)`); err != nil {
		return fmt.Errorf("failed to index the preferred paths: %w", err)
	}
	return nil
}

// candidate is a way to start or end a path at a vertex, with the cost of
// traveling between the vertex and the snapped point.
type candidate struct {
	vertex int64
	cost   float64
}

// search finds the cheapest path between two positions, within margin
// meters of them. It returns nil if there is none in the area.
func search(ctx context.Context, db store.Querier, from, to position, cost, reverseCost string, margin float64) (*Path, error) {
	// The costs of the snapped edges, to cost the partial edges at the ends.
	edgeCosts := map[int64][2]float64{}
	rows, err := db.Query(ctx, `select id, `+cost+`, `+reverseCost+` from routing_edges where id = any($1)`,
		[]int64{from.edge.id, to.edge.id})
	if err != nil {
		return nil, fmt.Errorf("failed to cost the snapped edges: %w", err)
	}
	// The rows hold a connection of the pool until they're closed.
	defer rows.Close()
	for rows.Next() {
		var id int64
		var forward, backward float64
		if err := rows.Scan(&id, &forward, &backward); err != nil {
			return nil, err
		}
		edgeCosts[id] = [2]float64{forward, backward}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to cost the snapped edges: %w", err)
	}
	fromCosts, toCosts := edgeCosts[from.edge.id], edgeCosts[to.edge.id]
	fromFraction := from.dist / math.Max(from.edge.length(), 1e-9)
	toFraction := to.dist / math.Max(to.edge.length(), 1e-9)
	starts := []candidate{
		{from.edge.source, fromFraction * fromCosts[1]},
		{from.edge.target, (1 - fromFraction) * fromCosts[0]},
	}
	ends := []candidate{
		{to.edge.source, toFraction * toCosts[0]},
		{to.edge.target, (1 - toFraction) * toCosts[1]},
	}

	bounds, _ := geo.BoundsOf([]geo.Point{from.edge.points[0], to.edge.points[0]})
	bounds = padBounds(bounds, margin)
	edgesSQL := fmt.Sprintf(`select id, source, target, %s as cost, %s as reverse_cost
		from routing_edges where geom && ST_MakeEnvelope(%f, %f, %f, %f, 4326)`,
		cost, reverseCost, bounds.MinLng, bounds.MinLat, bounds.MaxLng, bounds.MaxLat)
	rows, err = db.Query(ctx, `select start_vid, end_vid, node, edge, agg_cost
		from pgr_dijkstra($1, $2::bigint[], $3::bigint[], directed => true)
		order by start_vid, end_vid, path_seq`,
		edgesSQL, []int64{starts[0].vertex, starts[1].vertex}, []int64{ends[0].vertex, ends[1].vertex})
	if err != nil {
		return nil, fmt.Errorf("failed to search for a path: %w", err)
	}
	defer rows.Close()
	type step struct{ node, edge int64 }
	type route struct {
		start, end int64
		steps      []step
		cost       float64
	}
	found := map[[2]int64]*route{}
	for rows.Next() {
		var start, end, node, edgeID int64
		var aggCost float64
		if err := rows.Scan(&start, &end, &node, &edgeID, &aggCost); err != nil {
			return nil, err
		}
		key := [2]int64{start, end}
		if found[key] == nil {
			found[key] = &route{start: start, end: end}
		}
		found[key].cost = aggCost
		if edgeID >= 0 {
			found[key].steps = append(found[key].steps, step{node, edgeID})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to search for a path: %w", err)
	}

	best := math.Inf(1)
	var bestRoute *route
	var bestStart, bestEnd candidate
	direct := false
	for _, s := range starts {
		for _, e := range ends {
			var r *route
			if s.vertex == e.vertex {
				// pgRouting returns no path from a vertex to itself.
				r = &route{start: s.vertex, end: e.vertex}
			} else if r = found[[2]int64{s.vertex, e.vertex}]; r == nil {
				continue
			}
			if total := s.cost + r.cost + e.cost; total < best {
				best, bestRoute, bestStart, bestEnd = total, r, s, e
			}
		}
	}
	if from.edge.id == to.edge.id {
		// Both points are on the same edge, so the path may stay on it.
		f := math.Abs(toFraction - fromFraction)
		c := f * fromCosts[0]
		if toFraction < fromFraction {
			c = f * fromCosts[1]
		}
		if c <= best {
			best, direct = c, true
		}
	}
	if math.IsInf(best, 1) {
		return nil, nil
	}

	b := &pathBuilder{}
	if direct {
		b.add(from.edge.piece(from.dist, to.dist), from.edge.terrain)
		return b.path(), nil
	}
	// The path from the snapped point to the first vertex.
	b.add(from.edge.piece(from.dist, from.edge.vertexDist(bestStart.vertex)), from.edge.terrain)
	ids := make([]int64, len(bestRoute.steps))
	for i, s := range bestRoute.steps {
		ids[i] = s.edge
	}
	edges, err := loadEdges(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	for _, s := range bestRoute.steps {
		e := edges[s.edge]
		if e == nil {
			return nil, fmt.Errorf("edge %d of the path is missing", s.edge)
		}
		if s.node == e.source {
			b.add(e.piece(0, e.length()), e.terrain)
		} else {
			b.add(e.piece(e.length(), 0), e.terrain)
		}
	}
	// The path from the last vertex to the snapped point.
	b.add(to.edge.piece(to.edge.vertexDist(bestEnd.vertex), to.dist), to.edge.terrain)
	return b.path(), nil
}

func loadEdges(ctx context.Context, db store.Querier, ids []int64) (map[int64]*edge, error) {
	rows, err := db.Query(ctx, `select `+edgeColumns+` from routing_edges where id = any($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load the path's edges: %w", err)
	}
	defer rows.Close()
	edges := map[int64]*edge{}
	for rows.Next() {
		e, err := scanEdge(rows)
		if err != nil {
			return nil, err
		}
		edges[e.id] = e
	}
	return edges, rows.Err()
}

// vertexDist returns the distance along the edge to one of its vertices.
func (e *edge) vertexDist(vertex int64) float64 {
	if vertex == e.source {
		return 0
	}
	return e.length()
}

// piece returns the points of the edge between two distances along it, in
// the direction from the first to the second, with interpolated ends.
func (e *edge) piece(from, to float64) []geo.Point {
	reverse := from > to
	lo, hi := math.Min(from, to), math.Max(from, to)
	points := []geo.Point{e.at(lo)}
	for i, d := range e.dists {
		if d > lo && d < hi {
			points = append(points, e.points[i])
		}
	}
	points = append(points, e.at(hi))
	if reverse {
		for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
			points[i], points[j] = points[j], points[i]
		}
	}
	return points
}

// at returns the point at a distance along the edge.
func (e *edge) at(dist float64) geo.Point {
	for i := 1; i < len(e.points); i++ {
		if dist <= e.dists[i] {
			span := e.dists[i] - e.dists[i-1]
			if span <= 0 {
				return e.points[i]
			}
			return geo.Interpolate(e.points[i-1], e.points[i], (dist-e.dists[i-1])/span)
		}
	}
	return e.points[len(e.points)-1]
}

// pathBuilder joins pieces of edges into a path.
type pathBuilder struct {
	points   []geo.Point
	terrains []geo.Terrain
}

func (b *pathBuilder) add(points []geo.Point, terrain geo.Terrain) {
	for _, p := range points {
		if n := len(b.points); n > 0 && b.points[n-1].Lat == p.Lat && b.points[n-1].Lng == p.Lng {
			// The segment after a joining point belongs to the new piece.
			b.terrains[n-1] = terrain
			continue
		}
		b.points = append(b.points, p)
		b.terrains = append(b.terrains, terrain)
	}
}

func (b *pathBuilder) path() *Path {
	points, terrains := b.points, b.terrains
	if len(points) == 1 {
		// The points snapped to the same place.
		points = append(points, points[0])
		terrains = append(terrains, terrains[0])
	}
	terrains[len(terrains)-1] = geo.TerrainUnknown
	return &Path{Points: points, Terrains: terrains}
}

// padBounds grows bounds by about padding meters on each side.
func padBounds(b geo.Bounds, padding float64) geo.Bounds {
	midLat := (b.MinLat + b.MaxLat) / 2
	padLat := padding / metersPerDegreeLat
	padLng := padding / (metersPerDegreeLat * math.Max(0.01, math.Cos(midLat*math.Pi/180)))
	return geo.Bounds{
		MinLat: math.Max(-90, b.MinLat-padLat),
		MinLng: math.Max(-180, b.MinLng-padLng),
		MaxLat: math.Min(90, b.MaxLat+padLat),
		MaxLng: math.Min(180, b.MaxLng+padLng),
	}
}
