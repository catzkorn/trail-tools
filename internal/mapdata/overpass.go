package mapdata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/catzkorn/trail-tools/internal/geo"
)

// Element is an OpenStreetMap node or way returned by Overpass.
type Element struct {
	Type string            `json:"type"`
	ID   int64             `json:"id"`
	Tags map[string]string `json:"tags"`
	// Set for nodes.
	Lat float64 `json:"lat"`
	Lng float64 `json:"lon"`
	// Set for ways queried with "out center".
	Center *struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lon"`
	} `json:"center"`
	// Set for ways queried with "out geom".
	Geometry []struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lon"`
	} `json:"geometry"`
}

// Location returns the location of a node, or the center of a way.
func (e *Element) Location() (geo.Point, bool) {
	switch {
	case e.Type == "node":
		return geo.Point{Lat: e.Lat, Lng: e.Lng}, true
	case e.Center != nil:
		return geo.Point{Lat: e.Center.Lat, Lng: e.Center.Lng}, true
	case len(e.Geometry) > 0:
		return geo.Point{Lat: e.Geometry[0].Lat, Lng: e.Geometry[0].Lng}, true
	}
	return geo.Point{}, false
}

// Points returns the geometry of a way.
func (e *Element) Points() []geo.Point {
	points := make([]geo.Point, len(e.Geometry))
	for i, g := range e.Geometry {
		points[i] = geo.Point{Lat: g.Lat, Lng: g.Lng}
	}
	return points
}

// TrailHighways are the highway types considered trails.
var TrailHighways = []string{"path", "footway", "track", "bridleway", "steps", "cycleway"}

// FindTrails returns the ways that could be run on within radius meters of
// center, with their geometry.
func (c *Client) FindTrails(ctx context.Context, center geo.Point, radius float64) ([]Element, error) {
	query := fmt.Sprintf(
		`[out:json][timeout:25];way(around:%.0f,%f,%f)[highway~"^(%s)$"];out tags geom;`,
		radius, center.Lat, center.Lng, strings.Join(TrailHighways, "|"),
	)
	return c.overpass(ctx, query)
}

// FindFeatures returns nodes and ways matching any of the tag filters, e.g.
// `["natural"="peak"]`, within radius meters of center. Ways are returned with
// their center.
func (c *Client) FindFeatures(ctx context.Context, center geo.Point, radius float64, filters []string) ([]Element, error) {
	var sb strings.Builder
	sb.WriteString("[out:json][timeout:25];(")
	for _, f := range filters {
		fmt.Fprintf(&sb, "nwr(around:%.0f,%f,%f)%s;", radius, center.Lat, center.Lng, f)
	}
	sb.WriteString(");out center tags;")
	return c.overpass(ctx, sb.String())
}

// overpass runs a query against each configured Overpass instance in turn
// until one succeeds. Public instances are frequently overloaded.
func (c *Client) overpass(ctx context.Context, query string) ([]Element, error) {
	var errs []error
	for _, endpoint := range c.conf.OverpassURLs {
		form := url.Values{"data": {query}}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		var resp struct {
			Elements []Element `json:"elements"`
			Remark   string    `json:"remark"`
		}
		err = c.do(req, &resp)
		if err == nil && strings.Contains(resp.Remark, "error") {
			err = errors.New(resp.Remark)
		}
		if err == nil {
			return resp.Elements, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		errs = append(errs, fmt.Errorf("%s: %w", endpoint, err))
	}
	return nil, fmt.Errorf("OpenStreetMap query failed on all Overpass instances, they may be overloaded; try again later or with a smaller radius: %w", errors.Join(errs...))
}
