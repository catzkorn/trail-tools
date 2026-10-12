package routing

import (
	"bufio"
	"cmp"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"slices"
	"time"

	"github.com/catzkorn/trail-tools/internal/elevation"
	"github.com/catzkorn/trail-tools/internal/geo"
	"github.com/catzkorn/trail-tools/internal/store"
	"github.com/qedus/osmpbf"
)

// WaysFromOSM returns the foot accessible ways among OpenStreetMap ways, such
// as ones from the Overpass API, with the nodes they use.
func WaysFromOSM(osmWays []*osmpbf.Way, nodes map[int64]geo.Point) ([]Way, *Nodes) {
	var ways []Way
	var ids []int64
	for _, w := range osmWays {
		if isFootWay(w.Tags) {
			ways = append(ways, wayFromOSM(w))
			ids = append(ids, w.NodeIDs...)
		}
	}
	slices.Sort(ids)
	ids = slices.Compact(ids)
	result := &Nodes{}
	for _, id := range ids {
		if p, ok := nodes[id]; ok {
			result.IDs = append(result.IDs, id)
			result.Lats = append(result.Lats, p.Lat)
			result.Lngs = append(result.Lngs, p.Lng)
		}
	}
	return ways, result
}

// ImportWays loads ways into the routing graph as a region, with their
// elevations from the DEM, and returns the number of edges imported.
// Importing a region again replaces it.
func ImportWays(ctx context.Context, log *slog.Logger, db *store.DB, region string, bounds geo.Bounds, ways []Way, nodes *Nodes, dem *elevation.DEM) (int, error) {
	start := time.Now()
	elevations, err := dem.Elevations(ctx, nodes.Lats, nodes.Lngs)
	if err != nil {
		return 0, fmt.Errorf("failed to look up elevations: %w", err)
	}
	if bounds == (geo.Bounds{}) {
		bounds = geo.Bounds{MinLat: 90, MinLng: 180, MaxLat: -90, MaxLng: -180}
		for i := range nodes.IDs {
			if !math.IsNaN(nodes.Lats[i]) {
				bounds.MinLat, bounds.MaxLat = math.Min(bounds.MinLat, nodes.Lats[i]), math.Max(bounds.MaxLat, nodes.Lats[i])
				bounds.MinLng, bounds.MaxLng = math.Min(bounds.MinLng, nodes.Lngs[i]), math.Max(bounds.MaxLng, nodes.Lngs[i])
			}
		}
	}
	count, err := SaveRegion(ctx, db, region, bounds, BuildEdges(ways, nodes, elevations, bounds))
	if err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, fmt.Errorf("there are no foot accessible ways within the bounds")
	}
	log.InfoContext(ctx, "Imported the region", slog.String("region", region), slog.Int("edges", count), slog.Duration("duration", time.Since(start)))
	return count, nil
}

// waysMagic starts ways files.
const waysMagic = "TTWAYS1"

// coordinateScale is the precision coordinates are stored with, about 10 cm.
const coordinateScale = 1e6

// WriteWays writes ways and their nodes gzipped, compactly: IDs and
// coordinates are stored as differences to the ones before, and node
// references as differences between node indexes.
func WriteWays(w io.Writer, ways []Way, nodes *Nodes) error {
	zw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(zw)
	var buf [binary.MaxVarintLen64]byte
	varint := func(v int64) {
		n := binary.PutVarint(buf[:], v)
		_, _ = bw.Write(buf[:n])
	}
	str := func(s string) {
		varint(int64(len(s)))
		_, _ = bw.WriteString(s)
	}
	_, _ = bw.WriteString(waysMagic)

	varint(int64(len(nodes.IDs)))
	var prevID, prevLat, prevLng int64
	for i, id := range nodes.IDs {
		lat, lng := int64(math.Round(nodes.Lats[i]*coordinateScale)), int64(math.Round(nodes.Lngs[i]*coordinateScale))
		varint(id - prevID)
		varint(lat - prevLat)
		varint(lng - prevLng)
		prevID, prevLat, prevLng = id, lat, lng
	}

	sorted := slices.Clone(ways)
	slices.SortFunc(sorted, func(a, b Way) int { return cmp.Compare(a.ID, b.ID) })
	varint(int64(len(sorted)))
	prevID = 0
	prevIndex := int64(0)
	for _, way := range sorted {
		varint(way.ID - prevID)
		prevID = way.ID
		str(way.Highway)
		str(way.SacScale)
		str(string(way.Terrain))
		varint(int64(len(way.Nodes)))
		for _, id := range way.Nodes {
			index := int64(nodes.find(id))
			if index < 0 {
				return fmt.Errorf("way %d refers to the missing node %d", way.ID, id)
			}
			varint(index - prevIndex)
			prevIndex = index
		}
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	return zw.Close()
}

// ReadWays reads ways and their nodes written by WriteWays.
func ReadWays(r io.Reader) ([]Way, *Nodes, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read the ways: %w", err)
	}
	br := bufio.NewReader(zr)
	magic := make([]byte, len(waysMagic))
	if _, err := io.ReadFull(br, magic); err != nil || string(magic) != waysMagic {
		return nil, nil, errors.New("not a ways file")
	}
	var readErr error
	varint := func() int64 {
		v, err := binary.ReadVarint(br)
		if err != nil && readErr == nil {
			readErr = err
		}
		return v
	}
	count := func() int {
		n := varint()
		if n < 0 || n > 1<<26 {
			if readErr == nil {
				readErr = errors.New("invalid count")
			}
			return 0
		}
		return int(n)
	}
	str := func() string {
		b := make([]byte, min(count(), 1<<10))
		if _, err := io.ReadFull(br, b); err != nil && readErr == nil {
			readErr = err
		}
		return string(b)
	}

	n := count()
	nodes := &Nodes{IDs: make([]int64, n), Lats: make([]float64, n), Lngs: make([]float64, n)}
	var id, lat, lng int64
	for i := range n {
		id += varint()
		lat += varint()
		lng += varint()
		nodes.IDs[i], nodes.Lats[i], nodes.Lngs[i] = id, float64(lat)/coordinateScale, float64(lng)/coordinateScale
	}
	ways := make([]Way, count())
	id = 0
	index := int64(0)
	for i := range ways {
		id += varint()
		way := Way{ID: id, Highway: str(), SacScale: str(), Terrain: geo.Terrain(str())}
		way.Nodes = make([]int64, count())
		for j := range way.Nodes {
			index += varint()
			if index < 0 || index >= int64(n) {
				return nil, nil, errors.New("a way refers to a missing node")
			}
			way.Nodes[j] = nodes.IDs[index]
		}
		ways[i] = way
		if readErr != nil {
			break
		}
	}
	if readErr != nil {
		return nil, nil, fmt.Errorf("failed to read the ways: %w", readErr)
	}
	return ways, nodes, nil
}
