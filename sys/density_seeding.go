package sys

import (
	"math"

	"github.com/pwn-model/pwn/res"
)

// densitySeeding scales the seeds of attraction sources by their local
// density: each source cell's seed is set to its local occupancy fraction
// (the number of source cells within densityRadius, divided by the number
// of cells that fit in that radius -- so always in (0, 1]) raised to
// densityWeight. Shared by the systems computing attraction fields from
// trees (TreeAttraction, TreeAttractionOctants); see their docs for the
// meaning of DensityRadius and DensityWeight.
type densitySeeding struct {
	densityWeight float64

	// radiusCells is densityRadius expressed in grid cells. Left at 0 when
	// densityWeight is 0.
	radiusCells int

	// maxCount is the number of cells in a full (2*radiusCells+1) square
	// window, i.e. the largest localCount can ever be. Dividing by it turns
	// a raw neighbour count into an occupancy fraction in (0, 1]. Left at 0
	// when densityWeight is 0.
	maxCount float64

	// sat is a reused (width+1)*(height+1) summed-area-table buffer for
	// counting each source's local density, laid out the same column-major
	// way as res.Grid with an extra leading zero row/column. Left
	// unallocated when densityWeight is 0, since seedDensity never touches
	// it then.
	sat []float64
}

// newDensitySeeding creates a densitySeeding for a world of size ws.
//
// At densityWeight 0, occupancy^0 is 1 regardless of local density, so
// densityRadius is never consulted: it is not required to be valid, and
// the buffer that only the density count needs is not allocated.
func newDensitySeeding(densityWeight float64, densityRadius int, ws *res.WorldSize) densitySeeding {
	if densityWeight == 0 {
		return densitySeeding{densityWeight: densityWeight}
	}
	if densityRadius%ws.CellSize() != 0 {
		panic("DensityRadius of the dispersal submodel must be a multiple of the world's base cell size.")
	}
	radiusCells := densityRadius / ws.CellSize()
	return densitySeeding{
		densityWeight: densityWeight,
		radiusCells:   radiusCells,
		maxCount:      float64((2*radiusCells + 1) * (2*radiusCells + 1)),
		sat:           make([]float64, (ws.Width()+1)*(ws.Height()+1)),
	}
}

// seedDensity sets the seed of every source cell (marked in presence) in
// seeds to its local occupancy fraction raised to densityWeight. Leaves
// seeds untouched at densityWeight 0, where every seed is final at 1
// already (occupancy^0 is 1 for any occupancy).
//
// Computes a summed-area table of presence (counting each occupied cell as
// 1) into d.sat, so that localCount can answer a windowed count in O(1)
// instead of O(radius^2) per source, and sets the seed of every source
// from it. Both happen in the same pass over the grid: a source's
// localCount only needs the table up to column x+radiusCells+1, so the
// sources of each column are seeded as soon as that table column is
// complete, lagging radiusCells+1 columns behind, plus all remaining
// columns once the last table column is done.
func (d *densitySeeding) seedDensity(presence *res.Grid[bool], seeds *res.Grid[float64]) {
	if d.densityWeight == 0 {
		return
	}

	w, h := presence.Width(), presence.Height()
	stride := h + 1
	idx := func(x, y int) int { return x*stride + y }

	sat := d.sat
	for x := 0; x <= w; x++ {
		sat[idx(x, 0)] = 0
	}
	for y := 0; y <= h; y++ {
		sat[idx(0, y)] = 0
	}

	r := d.radiusCells
	next := 0 // next grid column to seed
	for x := 1; x <= w; x++ {
		for y := 1; y <= h; y++ {
			var v float64
			if presence.Get(x-1, y-1) {
				v = 1
			}
			sat[idx(x, y)] = v + sat[idx(x-1, y)] + sat[idx(x, y-1)] - sat[idx(x-1, y-1)]
		}

		last := x - 1 - r
		if x == w {
			last = w - 1
		}
		for ; next <= last; next++ {
			for y := range h {
				if presence.Get(next, y) {
					occupancy := d.localCount(presence, next, y) / d.maxCount
					seeds.Set(next, y, math.Pow(occupancy, d.densityWeight))
				}
			}
		}
	}
}

// localCount returns the number of source cells within densityRadius of
// (x, y), inclusive of (x, y) itself (so the result is always >= 1 when
// called on an actual source cell), via the summed-area table built by
// seedDensity.
func (d *densitySeeding) localCount(presence *res.Grid[bool], x, y int) float64 {
	w, h := presence.Width(), presence.Height()
	stride := h + 1
	idx := func(xx, yy int) int { return xx*stride + yy }

	r := d.radiusCells
	x1, x2 := max(x-r, 0), min(x+r, w-1)
	y1, y2 := max(y-r, 0), min(y+r, h-1)

	sat := d.sat
	return sat[idx(x2+1, y2+1)] - sat[idx(x1, y2+1)] - sat[idx(x2+1, y1)] + sat[idx(x1, y1)]
}
