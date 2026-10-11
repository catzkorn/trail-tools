package mapdata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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

// ErrRoutingBusy is returned when the BRouter server refuses requests because
// too many were made, as the public server does when its rate limit is hit.
var ErrRoutingBusy = errors.New("the routing service is busy; try again in a few minutes")

// routeRetryDelays are the delays before retrying a request that the BRouter
// server refused because it was busy.
var routeRetryDelays = []time.Duration{time.Second, 3 * time.Second}

// MaxAlternatives is the number of alternative routes BRouter can compute
// between two points, with alternative indexes 0 to MaxAlternatives-1.
const MaxAlternatives = 4

// Route computes a path between two points using BRouter. Alternative 0 is
// the best route for the profile, and alternatives 1 to MaxAlternatives-1
// are progressively different routes between the same points.
func (c *Client) Route(ctx context.Context, from, to geo.Point, profile string, alternative int) (*RoutedPath, error) {
	if alternative < 0 || alternative >= MaxAlternatives {
		return nil, fmt.Errorf("alternative must be between 0 and %d", MaxAlternatives-1)
	}
	q := url.Values{}
	q.Set("lonlats", fmt.Sprintf("%f,%f|%f,%f", from.Lng, from.Lat, to.Lng, to.Lat))
	q.Set("profile", profile)
	q.Set("alternativeidx", strconv.Itoa(alternative))
	q.Set("format", "geojson")
	for attempt := 0; ; attempt++ {
		resp, err := c.route(ctx, c.conf.BRouterURL+"?"+q.Encode())
		if !errors.Is(err, ErrRoutingBusy) || attempt >= len(routeRetryDelays) {
			return resp, err
		}
		select {
		case <-time.After(routeRetryDelays[attempt]):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (c *Client) route(ctx context.Context, url string) (*RoutedPath, error) {
	select {
	case c.brouter <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-c.brouter }()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	var resp brouterResponse
	if err := c.do(req, &resp); err != nil {
		statusErr := (*StatusError)(nil)
		if !errors.As(err, &statusErr) {
			return nil, fmt.Errorf("routing failed: %w", err)
		}
		switch {
		case statusErr.StatusCode == http.StatusTooManyRequests || statusErr.StatusCode == http.StatusServiceUnavailable ||
			strings.Contains(statusErr.Body, "retry later"):
			return nil, ErrRoutingBusy
		case statusErr.StatusCode < 500:
			// BRouter returns errors such as "from-position not mapped" as 4xx
			// responses with a plain text body.
			return nil, fmt.Errorf("routing failed: %s", statusErr.Body)
		default:
			return nil, fmt.Errorf("routing failed: %w", err)
		}
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
