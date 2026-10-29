package sys

import (
	"math"

	"github.com/pwn-model/pwn/res"
)

// propagateAttraction turns the seeded values of grid into a smooth
// attraction field by propagating each source's value outward, multiplying
// by decay per orthogonal step and decay^sqrt(2) per diagonal step, and
// keeping the best (max) value reaching each cell from any source. decay is
// derived from halfDistance (in meters) and the grid's own cell size, as
// decay = exp(-CellSize*ln(2) / halfDistance), so the field decays to half
// its source value at halfDistance meters regardless of grid resolution.
//
// This is an in-place, single-grid Gauss-Seidel relaxation (a discrete
// fast-sweeping method): two passes in opposite raster directions suffice,
// because the grid has no obstacles, so any shortest path from a source can
// be split into one forward-monotone and one backward-monotone segment, each
// fully resolved by one of the two passes.
func propagateAttraction(grid *res.Grid[float64], halfDistance float64) {
	w, h := grid.Width(), grid.Height()
	decay := math.Exp(-float64(grid.CellSize()) * math.Ln2 / halfDistance)

	// Offsets of the already-updated neighbours seen by a sweep moving in
	// increasing (forward) resp. decreasing (backward) x/y, paired with
	// their step decay factor.
	forward := [4][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}}
	backward := [4][2]int{{1, 1}, {1, 0}, {1, -1}, {0, 1}}
	decays := [4]float64{math.Pow(decay, math.Sqrt2), decay, math.Pow(decay, math.Sqrt2), decay}

	relax := func(x, y int, offsets [4][2]int) {
		best := grid.Get(x, y)
		for i, o := range offsets {
			nx, ny := x+o[0], y+o[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			if v := grid.Get(nx, ny) * decays[i]; v > best {
				best = v
			}
		}
		grid.Set(x, y, best)
	}

	for x := range w {
		for y := range h {
			relax(x, y, forward)
		}
	}
	for x := w - 1; x >= 0; x-- {
		for y := h - 1; y >= 0; y-- {
			relax(x, y, backward)
		}
	}
}
