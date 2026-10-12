package geo

import "math"

// metersPerDegreeLat is the approximate length of a degree of latitude.
const metersPerDegreeLat = 111320.0

// Grid divides bounds into roughly square cells, numbered row by row from
// the south west corner.
type Grid struct {
	Bounds Bounds
	// The number of rows and columns of cells.
	Rows, Cols       int
	cellLat, cellLng float64
	cellSize         float64
}

// NewGrid creates a grid of cells about cellSize meters wide over the bounds.
func NewGrid(bounds Bounds, cellSize float64) *Grid {
	midLat := (bounds.MinLat + bounds.MaxLat) / 2
	g := &Grid{
		Bounds:   bounds,
		cellSize: cellSize,
		cellLat:  cellSize / metersPerDegreeLat,
		cellLng:  cellSize / (metersPerDegreeLat * math.Max(0.01, math.Cos(midLat*math.Pi/180))),
	}
	g.Rows = int(math.Ceil((bounds.MaxLat - bounds.MinLat) / g.cellLat))
	g.Cols = int(math.Ceil((bounds.MaxLng - bounds.MinLng) / g.cellLng))
	return g
}

// Cell returns the cell containing p. ok is false if p is outside the grid.
func (g *Grid) Cell(p Point) (cell int, ok bool) {
	if p.Lat < g.Bounds.MinLat || p.Lat >= g.Bounds.MaxLat || p.Lng < g.Bounds.MinLng || p.Lng >= g.Bounds.MaxLng {
		return 0, false
	}
	row := int((p.Lat - g.Bounds.MinLat) / g.cellLat)
	col := int((p.Lng - g.Bounds.MinLng) / g.cellLng)
	return row*g.Cols + col, true
}

// Cells returns the set of cells a track passes through. Gaps between
// distant points, such as in low resolution polylines, are filled in.
func (g *Grid) Cells(points []Point) map[int]bool {
	cells := map[int]bool{}
	add := func(p Point) {
		if cell, ok := g.Cell(p); ok {
			cells[cell] = true
		}
	}
	for i, p := range points {
		if i == 0 {
			add(p)
			continue
		}
		prev := points[i-1]
		if Distance(prev, p) > g.cellSize/2 {
			for _, q := range Densify(prev, p, g.cellSize/2)[1:] {
				add(q)
			}
			continue
		}
		add(p)
	}
	return cells
}

// Center returns the center of a cell.
func (g *Grid) Center(cell int) (lat, lng float64) {
	row, col := cell/g.Cols, cell%g.Cols
	return g.Bounds.MinLat + (float64(row)+0.5)*g.cellLat, g.Bounds.MinLng + (float64(col)+0.5)*g.cellLng
}
