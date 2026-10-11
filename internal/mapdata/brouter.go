package mapdata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/catzkorn/trail-tools/internal/geo"
)

// BRouter profiles.
const (
	// ProfileHikingMountain prefers trails, including mountain trails.
	ProfileHikingMountain = "hiking-mountain"
	// ProfileShortest takes the shortest path.
	ProfileShortest = "shortest"
)

// RoutedPath is a path computed by BRouter.
type RoutedPath struct {
	// The points of the path, with elevations.
	Points []geo.Point
	// The terrain of each segment: Terrains[i] is the terrain between
	// Points[i] and Points[i+1]. It has the same length as Points.
	Terrains []geo.Terrain
}

type brouterResponse struct {
	Features []struct {
		Geometry struct {
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
		Properties struct {
			Messages [][]string `json:"messages"`
		} `json:"properties"`
	} `json:"features"`
}

// Route computes a path between two points using BRouter.
func (c *Client) Route(ctx context.Context, from, to geo.Point, profile string) (*RoutedPath, error) {
	q := url.Values{}
	q.Set("lonlats", fmt.Sprintf("%f,%f|%f,%f", from.Lng, from.Lat, to.Lng, to.Lat))
	q.Set("profile", profile)
	q.Set("alternativeidx", "0")
	q.Set("format", "geojson")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.conf.BRouterURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var resp brouterResponse
	if err := c.do(req, &resp); err != nil {
		if statusErr := (*StatusError)(nil); errors.As(err, &statusErr) && statusErr.StatusCode < 500 {
			// BRouter returns errors such as "from-position not mapped" as 4xx
			// responses with a plain text body.
			return nil, fmt.Errorf("routing failed: %s", statusErr.Body)
		}
		return nil, fmt.Errorf("routing failed: %w", err)
	}
	return parseBRouterResponse(&resp)
}

func parseBRouterResponse(resp *brouterResponse) (*RoutedPath, error) {
	if len(resp.Features) == 0 || len(resp.Features[0].Geometry.Coordinates) < 2 {
		return nil, errors.New("routing returned no path")
	}
	feature := resp.Features[0]
	path := &RoutedPath{}
	for _, coord := range feature.Geometry.Coordinates {
		if len(coord) < 2 {
			return nil, errors.New("invalid coordinate in routed path")
		}
		p := geo.Point{Lng: coord[0], Lat: coord[1]}
		if len(coord) >= 3 {
			p.Ele = coord[2]
		}
		path.Points = append(path.Points, p)
	}

	// Each message describes a stretch of the path with the same way tags. The
	// distance column is the length of the stretch. Map each segment of the
	// path to the stretch containing its midpoint.
	type stretch struct {
		end     float64
		terrain geo.Terrain
	}
	var stretches []stretch
	var total float64
	if msgs := feature.Properties.Messages; len(msgs) > 1 {
		header := msgs[0]
		distCol, tagsCol := -1, -1
		for i, h := range header {
			switch h {
			case "Distance":
				distCol = i
			case "WayTags":
				tagsCol = i
			}
		}
		if distCol >= 0 && tagsCol >= 0 {
			for _, msg := range msgs[1:] {
				if len(msg) <= distCol || len(msg) <= tagsCol {
					continue
				}
				d, err := strconv.ParseFloat(msg[distCol], 64)
				if err != nil {
					continue
				}
				total += d
				stretches = append(stretches, stretch{end: total, terrain: geo.ClassifyTerrain(geo.ParseTags(msg[tagsCol]))})
			}
		}
	}
	path.Terrains = make([]geo.Terrain, len(path.Points))
	if len(stretches) == 0 {
		return path, nil
	}
	dists := geo.CumulativeDistances(path.Points)
	scale := 1.0
	if pathLen := dists[len(dists)-1]; pathLen > 0 && total > 0 {
		scale = total / pathLen
	}
	s := 0
	for i := 0; i+1 < len(path.Points); i++ {
		mid := (dists[i] + dists[i+1]) / 2 * scale
		for s < len(stretches)-1 && mid > stretches[s].end {
			s++
		}
		path.Terrains[i] = stretches[s].terrain
	}
	return path, nil
}
