// Package elevation looks up ground elevations from a digital elevation model
// (DEM): public SRTM derived tiles, downloaded and cached as needed, so that
// lookups don't depend on a rate limited API.
package elevation

import (
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
	"github.com/catzkorn/trail-tools/internal/geo"
	"gitlab.com/greyxor/slogor"
)

// DefaultURL is the base URL of the elevation tiles: SRTM derived 1x1 degree
// tiles in the HGT format from the AWS Terrain Tiles open dataset.
const DefaultURL = "https://s3.amazonaws.com/elevation-tiles-prod/skadi"

// DefaultCacheDir is the default directory elevation tiles are cached in.
var DefaultCacheDir = filepath.Join(os.TempDir(), "trail-tools-dem")

// maxCachedTiles is the number of decoded tiles kept in memory. A tile is up
// to 26 MB, and most routes cover one or two.
const maxCachedTiles = 3

// demVoid marks missing data in HGT tiles.
const demVoid = -32768

// DEM looks up ground elevations from HGT tiles, downloading them to a cache
// directory as needed, and keeping the most recently used ones decoded in
// memory. It is safe for concurrent use.
type DEM struct {
	baseURL  string
	cacheDir string
	client   *http.Client

	// mu serialises loading tiles, so concurrent lookups don't download or
	// decode the same tile twice, and guards the fields below.
	mu sync.Mutex
	// An optional durable cache of the tiles, e.g. a Cloud Storage bucket,
	// for when the cache directory doesn't outlive the process.
	store blob.Store
	log   *slog.Logger
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
// client, or the default client if it's nil.
func NewDEM(baseURL, cacheDir string, client *http.Client) *DEM {
	if client == nil {
		client = http.DefaultClient
	}
	return &DEM{baseURL: baseURL, cacheDir: cacheDir, client: client}
}

// SetTileStore makes the DEM look for tiles missing from the cache directory
// in store before downloading them, and add the tiles it downloads to it.
// Failing to read or write the store only logs a warning, as the tiles can
// be downloaded instead.
func (d *DEM) SetTileStore(store blob.Store, log *slog.Logger) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.store, d.log = store, log
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
	path := filepath.Join(d.cacheDir, name+".hgt.gz")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := d.fetch(ctx, name, path); err != nil {
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

// tileKey returns the key of a tile in the tile store, following the tile
// server's paths, e.g. skadi/N46/N46E007.hgt.gz.
func tileKey(name string) string {
	return "skadi/" + name[:3] + "/" + name + ".hgt.gz"
}

// fetch writes the tile to path in the cache directory: from the tile store
// if it has it, or else downloaded and added to the tile store. An empty file
// or object records that the tile doesn't exist.
func (d *DEM) fetch(ctx context.Context, name, path string) error {
	if err := os.MkdirAll(d.cacheDir, 0o755); err != nil {
		return err
	}
	key := tileKey(name)
	if d.store != nil {
		data, err := d.store.Get(ctx, key)
		if err == nil {
			return writeFile(path, data)
		}
		if !errors.Is(err, blob.ErrNotFound) && ctx.Err() == nil {
			d.log.WarnContext(ctx, "Failed to read an elevation tile from the tile store, downloading it",
				slog.String("tile", name), slogor.Err(err))
		}
	}
	data, err := d.download(ctx, name)
	if err != nil {
		return err
	}
	if err := writeFile(path, data); err != nil {
		return err
	}
	if d.store != nil {
		if err := d.store.Put(ctx, key, data, "application/gzip"); err != nil && ctx.Err() == nil {
			d.log.WarnContext(ctx, "Failed to add an elevation tile to the tile store",
				slog.String("tile", name), slogor.Err(err))
		}
	}
	return nil
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
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to download elevation tile %s: %w", name, err)
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
