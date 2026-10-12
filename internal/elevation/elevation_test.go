package elevation_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/catzkorn/trail-tools/internal/elevation"
	"github.com/catzkorn/trail-tools/internal/geo"
)

func writeTile(t *testing.T, dir, name string, size int, sample func(row, col int) int16) {
	t.Helper()
	var raw bytes.Buffer
	for row := range size {
		for col := range size {
			if err := binary.Write(&raw, binary.BigEndian, sample(row, col)); err != nil {
				t.Fatal(err)
			}
		}
	}
	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	if _, err := w.Write(raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), gz.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDEM(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A tile rising 100m per sample to the east, from 0 at the west edge,
	// with a void in the north east corner. Rows go south from the north.
	writeTile(t, dir, "N46E007.hgt.gz", 3, func(row, col int) int16 {
		if row == 0 && col == 2 {
			return -32768
		}
		return int16(col * 100)
	})
	// An empty file records a missing tile, e.g. over the sea.
	if err := os.WriteFile(filepath.Join(dir, "N00E000.hgt.gz"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	// The URL is never used, as the tiles are cached.
	dem := elevation.NewDEM("http://invalid.test", dir)
	elevations, err := dem.Elevations(context.Background(), []float64{46.5, 46.25, 0.5}, []float64{7.25, 7.75, 0.5})
	if err != nil {
		t.Fatalf("Elevations: %v", err)
	}
	if math.Abs(elevations[0]-50) > 0.01 {
		t.Errorf("got %v, want 50 a quarter of the way east", elevations[0])
	}
	if math.Abs(elevations[1]-150) > 0.01 {
		t.Errorf("got %v, want 150 three quarters of the way east", elevations[1])
	}
	if !math.IsNaN(elevations[2]) {
		t.Errorf("got %v for a missing tile, want NaN", elevations[2])
	}
}

func TestPatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A tile of 11 samples a side, 0.1 degrees apart, rising 10 m per
	// sample to the east and 1 m per sample to the south.
	writeTile(t, dir, "N46E007.hgt.gz", 11, func(row, col int) int16 {
		return int16(1000 + col*10 + row)
	})
	ctx := context.Background()
	dem := elevation.NewDEM("http://invalid.test", dir)
	patch, err := dem.Patch(ctx, geo.Bounds{MinLat: 46.35, MinLng: 7.25, MaxLat: 46.62, MaxLng: 7.5})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	// The patch is aligned with the samples, covering the bounds.
	if b := patch.Bounds(); math.Abs(b.MinLat-46.3) > 1e-9 || math.Abs(b.MaxLat-46.7) > 1e-9 ||
		math.Abs(b.MinLng-7.2) > 1e-9 || math.Abs(b.MaxLng-7.5) > 1e-9 {
		t.Errorf("got patch bounds %+v", b)
	}
	var buf bytes.Buffer
	if _, err := patch.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	read, err := elevation.ReadPatch(&buf)
	if err != nil {
		t.Fatalf("ReadPatch: %v", err)
	}
	if _, err := elevation.ReadPatch(bytes.NewReader([]byte("not a patch"))); err == nil {
		t.Error("ReadPatch of garbage succeeded")
	}

	// With the patch, points in it are looked up without the tile, and
	// others from the tiles.
	empty := t.TempDir()
	withPatch := elevation.NewDEM("http://invalid.test", empty)
	withPatch.AddPatch(read)
	got, err := withPatch.Elevations(ctx, []float64{46.45, 46.6}, []float64{7.35, 7.3})
	if err != nil {
		t.Fatalf("Elevations: %v", err)
	}
	want, err := dem.Elevations(ctx, []float64{46.45, 46.6}, []float64{7.35, 7.3})
	if err != nil {
		t.Fatal(err)
	}
	for i := range got {
		if math.Abs(got[i]-want[i]) > 1e-6 {
			t.Errorf("point %d: got %v from the patch, want %v", i, got[i], want[i])
		}
	}
	// Outside the patch, the tile is downloaded, which fails here.
	if _, err := withPatch.Elevations(ctx, []float64{46.9}, []float64{7.9}); err == nil {
		t.Error("Elevations outside the patch didn't use the tiles")
	}
}
