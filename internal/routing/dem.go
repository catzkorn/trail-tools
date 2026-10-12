package routing

import (
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
)

// DefaultDEMURL is the base URL of the elevation tiles: SRTM derived 1x1
// degree tiles in the HGT format from the AWS Terrain Tiles open dataset.
const DefaultDEMURL = "https://s3.amazonaws.com/elevation-tiles-prod/skadi"

// DefaultDEMCache is the default directory elevation tiles are cached in.
var DefaultDEMCache = filepath.Join(os.TempDir(), "trail-tools-dem")

// demVoid marks missing data in HGT tiles.
const demVoid = -32768

// DEM looks up ground elevations from HGT tiles, downloading them to a cache
// directory as needed.
type DEM struct {
	baseURL  string
	cacheDir string
	client   *http.Client
}

// NewDEM creates a DEM that downloads tiles from baseURL into cacheDir.
func NewDEM(baseURL, cacheDir string) *DEM {
	return &DEM{baseURL: baseURL, cacheDir: cacheDir, client: http.DefaultClient}
}

// hgtTile is a 1x1 degree tile of elevations in meters, stored row by row
// from the north edge, with size samples on each side.
type hgtTile struct {
	lat, lng int
	size     int
	samples  []int16
}

// tileName returns the name of the tile whose south west corner is lat, lng,
// e.g. N46E007.
func tileName(lat, lng int) string {
	ns, ew := 'N', 'E'
	if lat < 0 {
		ns = 'S'
	}
	if lng < 0 {
		ew = 'W'
	}
	return fmt.Sprintf("%c%02d%c%03d", ns, abs(lat), ew, abs(lng))
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// tile loads the tile with the south west corner lat, lng. It returns nil if
// there is no tile, e.g. over the sea.
func (d *DEM) tile(ctx context.Context, lat, lng int) (*hgtTile, error) {
	name := tileName(lat, lng)
	path := filepath.Join(d.cacheDir, name+".hgt.gz")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := d.download(ctx, name, path); err != nil {
			return nil, err
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() == 0 {
		// An empty file records that the tile doesn't exist.
		return nil, nil
	}
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read tile %s: %w", name, err)
	}
	data, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("failed to read tile %s: %w", name, err)
	}
	size := int(math.Round(math.Sqrt(float64(len(data) / 2))))
	if size < 2 || size*size*2 != len(data) {
		return nil, fmt.Errorf("tile %s has an unexpected size of %d bytes", name, len(data))
	}
	t := &hgtTile{lat: lat, lng: lng, size: size, samples: make([]int16, size*size)}
	for i := range t.samples {
		t.samples[i] = int16(binary.BigEndian.Uint16(data[2*i:]))
	}
	return t, nil
}

func (d *DEM) download(ctx context.Context, name, path string) error {
	if err := os.MkdirAll(d.cacheDir, 0o755); err != nil {
		return err
	}
	url := fmt.Sprintf("%s/%s/%s.hgt.gz", d.baseURL, name[:3], name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download elevation tile %s: %w", name, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound, http.StatusForbidden:
		// There is no tile, e.g. over the sea.
		return os.WriteFile(path, nil, 0o644)
	default:
		return fmt.Errorf("failed to download elevation tile %s: status %d", name, resp.StatusCode)
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("failed to download elevation tile %s: %w", name, err)
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// elevation returns the elevation at a point in the tile by bilinear
// interpolation, or NaN if there is no data.
func (t *hgtTile) elevation(lat, lng float64) float64 {
	// Rows go south from the north edge, columns east from the west edge.
	y := (float64(t.lat+1) - lat) * float64(t.size-1)
	x := (lng - float64(t.lng)) * float64(t.size-1)
	row := min(max(int(y), 0), t.size-2)
	col := min(max(int(x), 0), t.size-2)
	fy, fx := y-float64(row), x-float64(col)
	corners := [4]int16{
		t.samples[row*t.size+col],
		t.samples[row*t.size+col+1],
		t.samples[(row+1)*t.size+col],
		t.samples[(row+1)*t.size+col+1],
	}
	weights := [4]float64{(1 - fx) * (1 - fy), fx * (1 - fy), (1 - fx) * fy, fx * fy}
	var sum, total float64
	for i, v := range corners {
		if v == demVoid {
			continue
		}
		sum += float64(v) * weights[i]
		total += weights[i]
	}
	if total == 0 {
		return math.NaN()
	}
	return sum / total
}

// Elevations returns the elevation of each point, loading one tile at a time
// to limit memory use. Elevations are NaN where there is no data.
func (d *DEM) Elevations(ctx context.Context, lats, lngs []float64) ([]float64, error) {
	type key struct{ lat, lng int }
	byTile := map[key][]int{}
	for i := range lats {
		k := key{int(math.Floor(lats[i])), int(math.Floor(lngs[i]))}
		byTile[k] = append(byTile[k], i)
	}
	elevations := make([]float64, len(lats))
	for k, idxs := range byTile {
		t, err := d.tile(ctx, k.lat, k.lng)
		if err != nil {
			return nil, err
		}
		for _, i := range idxs {
			elevations[i] = math.NaN()
			if t != nil {
				elevations[i] = t.elevation(lats[i], lngs[i])
			}
		}
	}
	return elevations, nil
}
