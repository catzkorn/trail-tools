package mapdata

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/catzkorn/trail-tools/internal/geo"
)

// maxElevationBatch is the maximum number of points per Open-Meteo request.
const maxElevationBatch = 100

// Elevations looks up the ground elevation of each point in meters using the
// Copernicus DEM via Open-Meteo.
func (c *Client) Elevations(ctx context.Context, points []geo.Point) ([]float64, error) {
	elevations := make([]float64, 0, len(points))
	for start := 0; start < len(points); start += maxElevationBatch {
		batch := points[start:min(start+maxElevationBatch, len(points))]
		lats := make([]string, len(batch))
		lngs := make([]string, len(batch))
		for i, p := range batch {
			lats[i] = strconv.FormatFloat(p.Lat, 'f', 6, 64)
			lngs[i] = strconv.FormatFloat(p.Lng, 'f', 6, 64)
		}
		q := url.Values{}
		q.Set("latitude", strings.Join(lats, ","))
		q.Set("longitude", strings.Join(lngs, ","))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.conf.ElevationURL+"?"+q.Encode(), nil)
		if err != nil {
			return nil, err
		}
		var resp struct {
			Elevation []float64 `json:"elevation"`
		}
		if err := c.do(req, &resp); err != nil {
			return nil, fmt.Errorf("elevation lookup failed: %w", err)
		}
		if len(resp.Elevation) != len(batch) {
			return nil, fmt.Errorf("elevation lookup returned %d values for %d points", len(resp.Elevation), len(batch))
		}
		elevations = append(elevations, resp.Elevation...)
	}
	return elevations, nil
}

// FillElevations sets the elevation of every point from the DEM. To limit the
// number of requests, at most maxLookups points are looked up and elevations in
// between are interpolated by distance.
func (c *Client) FillElevations(ctx context.Context, points []geo.Point, maxLookups int) error {
	if len(points) == 0 {
		return nil
	}
	idxs := geo.Downsample(len(points), maxLookups)
	lookup := make([]geo.Point, len(idxs))
	for i, idx := range idxs {
		lookup[i] = points[idx]
	}
	elevations, err := c.Elevations(ctx, lookup)
	if err != nil {
		return err
	}
	dists := geo.CumulativeDistances(points)
	for k := 0; k < len(idxs); k++ {
		points[idxs[k]].Ele = elevations[k]
		if k == 0 {
			continue
		}
		from, to := idxs[k-1], idxs[k]
		span := dists[to] - dists[from]
		for i := from + 1; i < to; i++ {
			f := 0.0
			if span > 0 {
				f = (dists[i] - dists[from]) / span
			}
			points[i].Ele = elevations[k-1] + (elevations[k]-elevations[k-1])*f
		}
	}
	return nil
}
