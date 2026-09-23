package maps

import (
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.RegisterObserver[TreeAttraction]()
}

// TreeAttraction map observer.
type TreeAttraction struct {
	DamagedTrees bool `yaml:"damaged_trees"`

	grid   res.Grid[float64]
	values []float64
}

// Initialize the observer.
func (o *TreeAttraction) Initialize(w *ecs.World) {
	if o.DamagedTrees {
		o.grid = ecs.GetResource[res.DamagedTreeAttraction](w).Grid
	} else {
		o.grid = ecs.GetResource[res.HealthyTreeAttraction](w).Grid
	}
	o.values = make([]float64, o.grid.Width()*o.grid.Height())
}

// Update the observer.
func (o *TreeAttraction) Update(_ *ecs.World) {}

// Dims returns the matrix dimensions.
func (o *TreeAttraction) Dims() (int, int) {
	return o.grid.Width(), o.grid.Height()
}

// Values for the current model tick, in row-major order (i.e. idx = row*ncols + col).
func (o *TreeAttraction) Values(_ *ecs.World) []float64 {
	w, h := o.grid.Width(), o.grid.Height()
	for x := range w {
		for y := range h {
			o.values[y*w+x] = math.Log(o.grid.Get(x, y))
		}
	}

	return o.values
}

// X axis coordinates.
func (o *TreeAttraction) X(c int) float64 {
	return float64(c * o.grid.CellSize())
}

// Y axis coordinates.
func (o *TreeAttraction) Y(r int) float64 {
	return float64(r * o.grid.CellSize())
}
