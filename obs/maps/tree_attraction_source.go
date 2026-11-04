package maps

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.RegisterObserver[TreeAttractionSourceMap]()
}

// TreeAttractionSourceMap reports the source cell of the attraction field of
// damaged or healthy trees (see DamagedTrees) per cell of the attraction
// grid, i.e. the tree cell each cell's attraction value was propagated from.
//
// To make cells sharing a source show up as patches of one color, with
// neighbouring patches clearly distinct, each source's cell index (see
// [res.Grid.Index]) is scrambled by an integer hash into a pseudo-random
// value in [0, 1). Plot it with min 0, max 1 and a gradient with many
// distinct colors, e.g. turbo. Cells without any source are reported as -1,
// below that range, so they get the gradient's lowest color. Unlike its
// sibling Julia implementation, which reports NaN there, as plot.Image's
// gradients have no defined color for NaN.
type TreeAttractionSourceMap struct {
	DamagedTrees bool `yaml:"damaged_trees" config:"optional"`

	grid   res.Grid[uint32]
	values []float64
}

// Initialize the observer.
func (o *TreeAttractionSourceMap) Initialize(w *ecs.World) {
	if o.DamagedTrees {
		o.grid = ecs.GetResource[res.DamagedTreeAttraction](w).Source
	} else {
		o.grid = ecs.GetResource[res.HealthyTreeAttraction](w).Source
	}
	o.values = make([]float64, o.grid.Width()*o.grid.Height())
}

// Update the observer.
func (o *TreeAttractionSourceMap) Update(_ *ecs.World) {}

// Dims returns the matrix dimensions.
func (o *TreeAttractionSourceMap) Dims() (int, int) {
	return o.grid.Width(), o.grid.Height()
}

// Values for the current model tick, in row-major order (i.e. idx = row*ncols + col).
func (o *TreeAttractionSourceMap) Values(_ *ecs.World) []float64 {
	w, h := o.grid.Width(), o.grid.Height()
	for x := range w {
		for y := range h {
			source := o.grid.Get(x, y)
			if source == res.NoCell {
				o.values[y*w+x] = -1
			} else {
				o.values[y*w+x] = hashToUnit(source)
			}
		}
	}

	return o.values
}

// X axis coordinates.
func (o *TreeAttractionSourceMap) X(c int) float64 {
	return float64(c * o.grid.CellSize())
}

// Y axis coordinates.
func (o *TreeAttractionSourceMap) Y(r int) float64 {
	return float64(r * o.grid.CellSize())
}

// hashToUnit maps a cell index to a pseudo-random float64 in [0, 1), via
// the SplitMix64 finalizer (so that consecutive indices, i.e. neighbouring
// cells, get unrelated values) and the top 53 bits of the result. Hashes
// index+1, so that a cell gets the same value as in the sibling Julia
// implementation, whose cell indices are 1-based.
func hashToUnit(index uint32) float64 {
	z := uint64(index) + 1 + 0x9e3779b97f4a7c15
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31
	return float64(z>>11) * 0x1p-53
}
