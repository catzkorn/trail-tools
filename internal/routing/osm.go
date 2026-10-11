package routing

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"slices"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/qedus/osmpbf"
)

// footHighways are the highway types that can be run on. Busier roads are
// included so routes can connect trails, but cost more to route along.
var footHighways = map[string]bool{
	"path": true, "footway": true, "track": true, "steps": true, "bridleway": true,
	"pedestrian": true, "living_street": true, "cycleway": true, "service": true,
	"residential": true, "unclassified": true, "road": true,
	"tertiary": true, "tertiary_link": true, "secondary": true, "secondary_link": true,
	"primary": true, "primary_link": true,
}

// isFootWay reports whether a way can be run on, from its tags.
func isFootWay(tags map[string]string) bool {
	if !footHighways[tags["highway"]] || tags["area"] == "yes" {
		return false
	}
	foot := tags["foot"]
	switch foot {
	case "no", "private", "use_sidepath":
		return false
	case "yes", "designated", "permissive":
		return true
	}
	switch tags["access"] {
	case "no", "private":
		return false
	}
	return true
}

// Way is a foot accessible OpenStreetMap way.
type Way struct {
	ID    int64
	Nodes []int64
	// The tags that affect routing.
	Highway, Surface, SacScale string
	Terrain                    geo.Terrain
}

func wayFromOSM(w *osmpbf.Way) Way {
	return Way{
		ID:       w.ID,
		Nodes:    w.NodeIDs,
		Highway:  w.Tags["highway"],
		Surface:  w.Tags["surface"],
		SacScale: w.Tags["sac_scale"],
		Terrain:  geo.ClassifyTerrain(w.Tags),
	}
}

// Nodes are the coordinates of OpenStreetMap nodes, sorted by ID.
type Nodes struct {
	IDs        []int64
	Lats, Lngs []float64
}

// find returns the index of a node, or -1 if it is missing.
func (n *Nodes) find(id int64) int {
	i, ok := slices.BinarySearch(n.IDs, id)
	if !ok {
		return -1
	}
	return i
}

// ReadPBF reads the foot accessible ways of an OpenStreetMap PBF extract and
// the coordinates of their nodes. The extract is read twice, as nodes come
// before the ways that use them, so open is called twice.
func ReadPBF(ctx context.Context, open func() (io.ReadCloser, error)) ([]Way, *Nodes, error) {
	var ways []Way
	err := scanPBF(ctx, open, func(v any) {
		if w, ok := v.(*osmpbf.Way); ok && isFootWay(w.Tags) {
			ways = append(ways, wayFromOSM(w))
		}
	})
	if err != nil {
		return nil, nil, err
	}
	var ids []int64
	for _, w := range ways {
		ids = append(ids, w.Nodes...)
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	nodes := &Nodes{IDs: ids, Lats: make([]float64, len(ids)), Lngs: make([]float64, len(ids))}
	for i := range ids {
		nodes.Lats[i], nodes.Lngs[i] = math.NaN(), math.NaN()
	}
	err = scanPBF(ctx, open, func(v any) {
		if n, ok := v.(*osmpbf.Node); ok {
			if i := nodes.find(n.ID); i >= 0 {
				nodes.Lats[i], nodes.Lngs[i] = n.Lat, n.Lon
			}
		}
	})
	if err != nil {
		return nil, nil, err
	}
	return ways, nodes, nil
}

func scanPBF(ctx context.Context, open func() (io.ReadCloser, error), handle func(any)) error {
	r, err := open()
	if err != nil {
		return err
	}
	defer r.Close()
	d := osmpbf.NewDecoder(r)
	if err := d.Start(runtime.GOMAXPROCS(0)); err != nil {
		return fmt.Errorf("failed to read the OpenStreetMap extract: %w", err)
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		v, err := d.Decode()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to read the OpenStreetMap extract: %w", err)
		}
		handle(v)
	}
}

// Edge is a section of a way between two junctions or ends.
type Edge struct {
	WayID          int64
	Source, Target int64
	Points         []geo.Point
	Highway        string
	Surface        string
	SacScale       string
	Terrain        geo.Terrain
}

// BuildEdges splits ways into edges at the nodes where they meet or end.
// Ways without any node inside bounds are left out, and so are nodes
// missing from the extract. Points get their elevation from elevations,
// which has one entry per node, NaN if unknown.
func BuildEdges(ways []Way, nodes *Nodes, elevations []float64, bounds geo.Bounds) []Edge {
	// Junctions are nodes used more than once, by different ways or twice by
	// the same way, as in a loop.
	uses := make([]uint8, len(nodes.IDs))
	for _, w := range ways {
		for _, id := range w.Nodes {
			if i := nodes.find(id); i >= 0 && uses[i] < 2 {
				uses[i]++
			}
		}
	}
	var edges []Edge
	for _, w := range ways {
		// The indexes of the way's nodes that have coordinates.
		var idxs []int
		inside := false
		for _, id := range w.Nodes {
			i := nodes.find(id)
			if i < 0 || math.IsNaN(nodes.Lats[i]) {
				continue
			}
			idxs = append(idxs, i)
			lat, lng := nodes.Lats[i], nodes.Lngs[i]
			if lat >= bounds.MinLat && lat <= bounds.MaxLat && lng >= bounds.MinLng && lng <= bounds.MaxLng {
				inside = true
			}
		}
		if !inside || len(idxs) < 2 {
			continue
		}
		start := 0
		for k := 1; k < len(idxs); k++ {
			if k < len(idxs)-1 && uses[idxs[k]] < 2 {
				continue
			}
			edge := Edge{
				WayID:    w.ID,
				Source:   nodes.IDs[idxs[start]],
				Target:   nodes.IDs[idxs[k]],
				Highway:  w.Highway,
				Surface:  w.Surface,
				SacScale: w.SacScale,
				Terrain:  w.Terrain,
			}
			for _, i := range idxs[start : k+1] {
				edge.Points = append(edge.Points, geo.Point{Lat: nodes.Lats[i], Lng: nodes.Lngs[i], Ele: elevations[i]})
			}
			edges = append(edges, edge)
			start = k
		}
	}
	return edges
}
