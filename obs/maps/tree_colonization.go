package maps

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
)

// TreeColonization reports the number of colonized trees per grid cell of
// CellSize (in meters), aggregated over the base tree grid.
//
// Mirrors the sys.Colonization system's density aggregation, but exposed as
// an observer.Matrix (github.com/mlange-42/ark-tools/observer) for plotting,
// e.g. with plot.Image from github.com/mlange-42/ark-pixel/plot.
type TreeColonization struct {
	// CellSize of the aggregation grid, in meters. Must be a multiple of
	// the world's base cell size.
	CellSize int

	coloFilter *ecs.Filter1[comp.Position]

	width, height int

	// unitsPerCell is the number of tree-grid units per aggregation-grid
	// cell, i.e. CellSize expressed in the world's base cell-size units.
	unitsPerCell int

	counts []float64
}

// Initialize the observer.
func (o *TreeColonization) Initialize(w *ecs.World) {
	o.coloFilter = o.coloFilter.New(w).With(ecs.C[comp.Colonized]())

	ws := ecs.GetResource[res.WorldSize](w)
	if o.CellSize%ws.CellSize() != 0 {
		panic("CellSize of the tree colonization map observer must be a multiple of the world's base cell size.")
	}
	o.unitsPerCell = o.CellSize / ws.CellSize()

	o.width, o.height = util.CeilDiv(ws.Width(), o.unitsPerCell), util.CeilDiv(ws.Height(), o.unitsPerCell)
	o.counts = make([]float64, o.width*o.height)
}

// Update the observer.
func (o *TreeColonization) Update(_ *ecs.World) {}

// Dims returns the matrix dimensions.
func (o *TreeColonization) Dims() (int, int) {
	return o.width, o.height
}

// Values for the current model tick, in row-major order (i.e. idx = row*ncols + col).
func (o *TreeColonization) Values(_ *ecs.World) []float64 {
	for i := range o.counts {
		o.counts[i] = 0
	}

	q := o.coloFilter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := positions[i]
			x, y := o.toCoords(pos.X, pos.Y)
			o.counts[y*o.width+x]++
		}
	}

	return o.counts
}

// X axis coordinates.
func (o *TreeColonization) X(c int) float64 {
	return float64(c * o.CellSize)
}

// Y axis coordinates.
func (o *TreeColonization) Y(r int) float64 {
	return float64(r * o.CellSize)
}

// toCoords calculates map-grid coords from tree grid coords.
func (o *TreeColonization) toCoords(x, y int) (int, int) {
	return x / o.unitsPerCell, y / o.unitsPerCell
}
