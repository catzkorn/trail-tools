// Package elevation looks up ground elevations from a digital elevation model
// (DEM): public SRTM derived tiles, downloaded and cached as needed, so that
// lookups don't depend on a rate limited API.
package elevation

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/catzkorn/trail-tools/internal/blob"
	"github.com/catzkorn/trail-tools/internal/errreport"
	"github.com/catzkorn/trail-tools/internal/geo"
)

// DefaultURL is the base URL of the elevation tiles: SRTM derived 1x1 degree
// tiles in the HGT format from the AWS Terrain Tiles open dataset.
const DefaultURL = "https://s3.amazonaws.com/elevation-tiles-prod/skadi"

// DefaultCacheDir is the default directory elevation tiles are cached in.
var DefaultCacheDir = filepath.Join(os.TempDir(), "trail-tools-dem")

// maxTileBytes is the size of the largest tiles, of 1 arc second samples
// (3601 a side), uncompressed.
const maxTileBytes = 3601 * 3601 * 2

// maxDownloadBytes limits the size of downloaded tiles, which are gzipped and
// so smaller than maxTileBytes.
const maxDownloadBytes = 32 << 20

// maxCachedTiles is the number of decoded tiles kept in memory. A tile is up
// to 26 MB, and most routes cover one or two.
const maxCachedTiles = 3

// demVoid marks missing data in HGT tiles.
const demVoid = -32768

// DEM looks up ground elevations from HGT tiles, downloading them to a cache
// directory, if it has one, as needed, and keeping the most recently used
// ones decoded in memory. It is safe for concurrent use.
type DEM struct {
	baseURL  string
	cacheDir string
	client   *http.Client

	// mu serialises loading tiles, so concurrent lookups don't download or
	// decode the same tile twice, and guards the fields below.
	mu sync.Mutex
	// An optional durable cache of the tiles, e.g. a Cloud Storage bucket,
	// for when the cache directory doesn't outlive the process.
	store  blob.Store
	report *errreport.Reporter
	// The most recently used tiles, most recent first. A nil tile records
	// that there is no tile, e.g. over the sea.
	tiles []cachedTile
	// Areas looked up without loading tiles.
	patches []*Patch
}

type cachedTile struct {
	lat, lng int
	tile     *hgtTile
}

// NewDEM creates a DEM that downloads tiles from baseURL into cacheDir with
// client, or the default client if it's nil. If cacheDir is empty, tiles
// aren't kept on disk, only the most recently used ones in memory.
func NewDEM(baseURL, cacheDir string, client *http.Client) *DEM {
	if client == nil {
		client = http.DefaultClient
	}
	return &DEM{baseURL: baseURL, cacheDir: cacheDir, client: client}
}

// SetTileStore makes the DEM look for tiles missing from the cache directory,
// or all tiles if it has none, in store before downloading them, and add the
// tiles it downloads to it. Failing to read or write the store is reported,
// but the tiles are downloaded instead.
func (d *DEM) SetTileStore(store blob.Store, report *errreport.Reporter) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.store, d.report = store, report
}

// cachedTile returns the tile with the south west corner lat, lng, loading it
// if it isn't in memory.
func (d *DEM) cachedTile(ctx context.Context, lat, lng int) (*hgtTile, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i, c := range d.tiles {
		if c.lat == lat && c.lng == lng {
			// Move it to the front.
			copy(d.tiles[1:i+1], d.tiles[:i])
			d.tiles[0] = c
			return c.tile, nil
		}
	}
	t, err := d.tile(ctx, lat, lng)
	if err != nil {
		return nil, err
	}
	if len(d.tiles) < maxCachedTiles {
		d.tiles = append(d.tiles, cachedTile{})
	}
	copy(d.tiles[1:], d.tiles)
	d.tiles[0] = cachedTile{lat: lat, lng: lng, tile: t}
	return t, nil
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
	data, err := d.tileData(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		// No data records that the tile doesn't exist.
		return nil, nil
	}
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to read tile %s: %w", name, err)
	}
	raw, err := io.ReadAll(io.LimitReader(zr, maxTileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read tile %s: %w", name, err)
	}
	size := int(math.Round(math.Sqrt(float64(len(raw) / 2))))
	if size < 2 || size*size*2 != len(raw) || len(raw) > maxTileBytes {
		return nil, fmt.Errorf("tile %s has an unexpected size of %d bytes", name, len(raw))
	}
	t := &hgtTile{lat: lat, lng: lng, size: size, samples: make([]int16, size*size)}
	for i := range t.samples {
		t.samples[i] = int16(binary.BigEndian.Uint16(raw[2*i:]))
	}
	return t, nil
}

// tileData returns the gzipped tile from the cache directory, if the DEM
// has one, or else fetches it and adds it to the cache directory. No data
// records that the tile doesn't exist.
func (d *DEM) tileData(ctx context.Context, name string) ([]byte, error) {
	if d.cacheDir == "" {
		return d.fetch(ctx, name)
	}
	path := filepath.Join(d.cacheDir, name+".hgt.gz")
	data, err := os.ReadFile(path)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return data, err
	}
	data, err = d.fetch(ctx, name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(d.cacheDir, 0o755); err != nil {
		return nil, err
	}
	if err := writeFile(path, data); err != nil {
		return nil, err
	}
	return data, nil
}

// tileKey returns the key of a tile in the tile store, following the tile
// server's paths, e.g. skadi/N46/N46E007.hgt.gz.
func tileKey(name string) string {
	return "skadi/" + name[:3] + "/" + name + ".hgt.gz"
}

// fetch returns the gzipped tile from the tile store if it has it, or else
// downloads it and adds it to the tile store. No data, or an empty object,
// records that the tile doesn't exist.
func (d *DEM) fetch(ctx context.Context, name string) ([]byte, error) {
	key := tileKey(name)
	if d.store != nil {
		data, err := d.store.Get(ctx, key)
		if err == nil {
			return data, nil
		}
		if !errors.Is(err, blob.ErrNotFound) && ctx.Err() == nil {
			d.report.Report(ctx, "Failed to read an elevation tile from the tile store, downloading it", err,
				slog.String("tile", name))
		}
	}
	data, err := d.download(ctx, name)
	if err != nil {
		return nil, err
	}
	if d.store != nil {
		if err := d.store.Put(ctx, key, data, "application/gzip"); err != nil && ctx.Err() == nil {
			d.report.Report(ctx, "Failed to add an elevation tile to the tile store", err,
				slog.String("tile", name))
		}
	}
	return data, nil
}

// download downloads the gzipped tile. It returns no data if there is no
// tile, e.g. over the sea.
func (d *DEM) download(ctx context.Context, name string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s.hgt.gz", d.baseURL, name[:3], name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download elevation tile %s: %w", name, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound, http.StatusForbidden:
		// There is no tile, e.g. over the sea.
		return nil, nil
	default:
		return nil, fmt.Errorf("failed to download elevation tile %s: status %d", name, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to download elevation tile %s: %w", name, err)
	}
	if len(data) > maxDownloadBytes {
		return nil, fmt.Errorf("elevation tile %s is larger than %d bytes", name, maxDownloadBytes)
	}
	return data, nil
}

// writeFile writes data to a temporary file and renames it to path, so
// readers never see a partial file.
func writeFile(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
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

// Elevations returns the elevation of each point, from the patches or else
// the tiles, loading one tile at a time to limit memory use. Elevations are
// NaN where there is no data.
func (d *DEM) Elevations(ctx context.Context, lats, lngs []float64) ([]float64, error) {
	elevations := make([]float64, len(lats))
	var rest []int
	for i := range lats {
		e, ok := d.patchElevation(lats[i], lngs[i])
		if !ok {
			rest = append(rest, i)
		}
		elevations[i] = e
	}
	if len(rest) == 0 {
		return elevations, nil
	}
	restLats := make([]float64, len(rest))
	restLngs := make([]float64, len(rest))
	for j, i := range rest {
		restLats[j], restLngs[j] = lats[i], lngs[i]
	}
	fromTiles, err := d.tileElevations(ctx, restLats, restLngs)
	if err != nil {
		return nil, err
	}
	for j, i := range rest {
		elevations[i] = fromTiles[j]
	}
	return elevations, nil
}

// tileElevations returns the elevation of each point from the tiles.
func (d *DEM) tileElevations(ctx context.Context, lats, lngs []float64) ([]float64, error) {
	type key struct{ lat, lng int }
	byTile := map[key][]int{}
	for i := range lats {
		k := key{int(math.Floor(lats[i])), int(math.Floor(lngs[i]))}
		byTile[k] = append(byTile[k], i)
	}
	elevations := make([]float64, len(lats))
	for k, idxs := range byTile {
		t, err := d.cachedTile(ctx, k.lat, k.lng)
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

// Points returns the elevation of each point. Points without data, e.g. over
// the sea, are at sea level.
func (d *DEM) Points(ctx context.Context, points []geo.Point) ([]float64, error) {
	lats := make([]float64, len(points))
	lngs := make([]float64, len(points))
	for i, p := range points {
		lats[i], lngs[i] = p.Lat, p.Lng
	}
	elevations, err := d.Elevations(ctx, lats, lngs)
	if err != nil {
		return nil, err
	}
	for i, e := range elevations {
		if math.IsNaN(e) {
			elevations[i] = 0
		}
	}
	return elevations, nil
}
