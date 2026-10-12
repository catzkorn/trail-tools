package routing_test

import (
	"bytes"
	"math"
	"slices"
	"testing"

	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/routing"
	"github.com/qedus/osmpbf"
)

func TestWaysRoundTrip(t *testing.T) {
	ways, nodes, _ := testGraph()
	var buf bytes.Buffer
	if err := routing.WriteWays(&buf, ways, nodes); err != nil {
		t.Fatalf("WriteWays: %v", err)
	}
	written := bytes.Clone(buf.Bytes())
	gotWays, gotNodes, err := routing.ReadWays(&buf)
	if err != nil {
		t.Fatalf("ReadWays: %v", err)
	}
	if !slices.Equal(gotNodes.IDs, nodes.IDs) {
		t.Errorf("got node IDs %v, want %v", gotNodes.IDs, nodes.IDs)
	}
	for i := range nodes.IDs {
		if math.Abs(gotNodes.Lats[i]-nodes.Lats[i]) > 1e-6 || math.Abs(gotNodes.Lngs[i]-nodes.Lngs[i]) > 1e-6 {
			t.Errorf("node %d at %v, %v, want %v, %v", nodes.IDs[i], gotNodes.Lats[i], gotNodes.Lngs[i], nodes.Lats[i], nodes.Lngs[i])
		}
	}
	byID := map[int64]routing.Way{}
	for _, w := range gotWays {
		byID[w.ID] = w
	}
	if len(gotWays) != len(ways) {
		t.Fatalf("got %d ways, want %d", len(gotWays), len(ways))
	}
	for _, want := range ways {
		got := byID[want.ID]
		if got.Highway != want.Highway || got.SacScale != want.SacScale || got.Terrain != want.Terrain || !slices.Equal(got.Nodes, want.Nodes) {
			t.Errorf("got way %+v, want %+v", got, want)
		}
	}
	// Garbage and truncated files fail.
	for _, data := range [][]byte{nil, []byte("not gzip"), written[:len(written)/2]} {
		if _, _, err := routing.ReadWays(bytes.NewReader(data)); err == nil {
			t.Errorf("ReadWays of %d bytes succeeded", len(data))
		}
	}
}

func TestWaysFromOSM(t *testing.T) {
	ways, nodes := routing.WaysFromOSM([]*osmpbf.Way{
		{ID: 1, Tags: map[string]string{"highway": "path", "sac_scale": "mountain_hiking"}, NodeIDs: []int64{10, 11}},
		{ID: 2, Tags: map[string]string{"highway": "motorway"}, NodeIDs: []int64{11, 12}},
		{ID: 3, Tags: map[string]string{"highway": "service", "service": "parking_aisle"}, NodeIDs: []int64{12, 13}},
		{ID: 4, Tags: map[string]string{"highway": "residential"}, NodeIDs: []int64{11, 14}},
	}, map[int64]geo.Point{10: {Lat: 1, Lng: 1}, 11: {Lat: 2, Lng: 2}, 12: {Lat: 3, Lng: 3}, 13: {Lat: 4, Lng: 4}, 14: {Lat: 5, Lng: 5}})
	var ids []int64
	for _, w := range ways {
		ids = append(ids, w.ID)
	}
	if !slices.Equal(ids, []int64{1, 4}) || ways[0].SacScale != "mountain_hiking" {
		t.Errorf("got ways %+v, want the path and the residential road", ways)
	}
	if !slices.Equal(nodes.IDs, []int64{10, 11, 14}) || nodes.Lats[2] != 5 {
		t.Errorf("got nodes %+v, want those of the foot ways", nodes)
	}
}
