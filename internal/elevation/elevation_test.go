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
