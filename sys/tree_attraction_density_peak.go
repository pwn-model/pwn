package sys

import (
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[TreeAttractionDensityPeak]()
}

// TreeAttractionDensityPeak system.
//
// A fourth alternative to [TreeAttraction] (see also [TreeAttractionSweep]
// and [TreeAttractionRecursive]), for a deterministic consumer that always
// moves towards whichever neighbouring cell has the highest attraction
// value. That rules out applying a density term *after* propagation the
// way the other alternatives do: raising the final field to a DensityWeight
// exponent is a uniform, strictly increasing transform, so by the chain
// rule it can never change a gradient's direction, nor which of several
// candidate cells has the highest value -- it only ever rescales the
// numbers, never a decision made from them.
//
// So this system makes the SEED fed into fillGrid depend on each source's
// own local tree density instead (see fillFromQuery). A denser cluster
// starts from a taller seed, so it can out-reach and win cells that a
// closer but sparser source would otherwise have claimed by
// max-relaxation: a real, comparison-relevant effect, decided before
// propagation ever compares sources against each other, rather than an
// inert transform applied after the fact.
//
// fillGrid itself keeps TreeAttraction's max combinator (never sum --
// see [TreeAttractionSum]'s doc comment for why summing across this
// function's 4-neighbour-per-step structure diverges), but decays
// multiplicatively instead of subtracting a fixed cost per step. Unlike
// subtraction, multiplication never drives a reachable cell to exactly
// zero, so there's no hard radius the way TreeAttraction/TreeAttractionSweep
// have -- and unlike sum, max can never exceed the largest seed anywhere in
// the grid (multiplying by a factor in (0,1) only ever shrinks a value), so
// this stays just as stable and bounded as TreeAttraction, for any Scale.
//
// Only one of TreeAttraction, TreeAttractionSweep, TreeAttractionRecursive,
// TreeAttractionDensityPeak, TreeAttractionSum or
// TreeAttractionDensityRecursive is meant to be active in a given config,
// since they all publish to the same [res.HealthyTreeAttraction] /
// [res.DamagedTreeAttraction] resources.
type TreeAttractionDensityPeak struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.

	// Scale is the e-folding decay length, in meters: a source's
	// contribution to a cell decay^chamfer_distance(cell, source) meters
	// away, where decay = exp(-CellSize/Scale).
	Scale int `yaml:"scale"`

	// DensityRadius is the radius, in meters, within which a source tree's
	// own same-type neighbours are counted towards its local density.
	// Independent of Scale: this is the "how clustered is this source"
	// scale, not the "how far does its seed reach" scale. Must be a
	// multiple of the world's base cell size. Unused, and not validated,
	// when DensityWeight is 0 -- see DensityWeight and fillFromQuery.
	DensityRadius int `yaml:"density_radius"`

	// DensityWeight is the exponent applied to a source's local occupancy
	// fraction (its same-type neighbour count within DensityRadius,
	// divided by the number of cells that fit in that radius -- so always
	// in (0, 1]) to scale its seed: 0 makes every source seed at 1
	// regardless of clustering (a plain single-nearest/strongest-source
	// decay field), 1 makes the seed scale linearly with local occupancy,
	// and >1 makes denser clusters reach disproportionately farther than
	// the same trees spread out. Since occupancy is normalized to (0, 1],
	// every seed -- and so, by fillGrid's max-relaxation, the whole
	// resulting field -- stays within (0, 1] regardless of DensityRadius
	// or how densely populated the world is.
	DensityWeight float64 `yaml:"density_weight"`

	timeRes ecs.Resource[res.Time]

	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	// decay is the per-orthogonal-cell-step decay factor derived from
	// Scale; a diagonal step uses decay^sqrt(2), mirroring
	// TreeAttraction's sqrt(2) diagonal cost.
	decay float64

	// densityRadiusCells is DensityRadius expressed in grid cells. Left at
	// its zero value when DensityWeight is 0.
	densityRadiusCells int

	// maxCount is the number of cells in a full (2*densityRadiusCells+1)
	// square window, i.e. the largest localCount can ever be. Dividing by
	// it turns a raw neighbour count into an occupancy fraction in (0, 1].
	// Left at its zero value when DensityWeight is 0.
	maxCount float64

	healthyAttraction res.Grid[float64]
	damagedAttraction res.Grid[float64]

	// presence is a reused 0/1 scratch grid for whichever type
	// (healthy/damaged) is currently being seeded. Left unallocated when
	// DensityWeight is 0, since fillFromQuery's fast path never touches it.
	presence res.Grid[float64]

	// sat is a reused (width+1)*(height+1) summed-area-table buffer for
	// counting each source's local density, laid out the same column-major
	// way as res.Grid with an extra leading zero row/column, as in
	// TreeAttractionSweep's boxBlur. Left unallocated when DensityWeight is
	// 0, since fillFromQuery's fast path never touches it.
	sat []float64
}

// Initialize the system.
func (s *TreeAttractionDensityPeak) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	s.decay = math.Exp(-float64(ws.CellSize()) / float64(s.Scale))

	s.healthyAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.damagedAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	ecs.AddResource(world, &res.HealthyTreeAttraction{Grid: s.healthyAttraction})
	ecs.AddResource(world, &res.DamagedTreeAttraction{Grid: s.damagedAttraction})

	// At DensityWeight 0, occupancy^0 is 1 regardless of local density (see
	// fillFromQuery's fast path), so DensityRadius is never consulted:
	// don't require it to be valid, and don't allocate the buffers that
	// only the density count needs.
	if s.DensityWeight != 0 {
		if s.DensityRadius%ws.CellSize() != 0 {
			panic("DensityRadius of the dispersal submodel must be a multiple of the world's base cell size.")
		}
		s.densityRadiusCells = s.DensityRadius / ws.CellSize()
		s.maxCount = float64((2*s.densityRadiusCells + 1) * (2*s.densityRadiusCells + 1))

		s.presence = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
		s.sat = make([]float64, (ws.Width()+1)*(ws.Height()+1))
	}
}

// Update the system.
func (s *TreeAttractionDensityPeak) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.calcAttraction()
	}
}

// Finalize the system.
func (s *TreeAttractionDensityPeak) Finalize(_ *ecs.World) {}

func (s *TreeAttractionDensityPeak) calcAttraction() {
	s.fillFromQuery(&s.healthyAttraction, s.filterHealthy)
	s.fillFromQuery(&s.damagedAttraction, s.filterDamaged)

	s.fillGrid(&s.healthyAttraction)
	s.fillGrid(&s.damagedAttraction)
}

// fillFromQuery seeds the attraction grid: every cell containing a
// matching tree is set to that source's own local occupancy fraction
// (localCount divided by maxCount, always in (0, 1]) raised to
// DensityWeight, everything else to 0. fillGrid then propagates these
// seeds outward.
func (s *TreeAttractionDensityPeak) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position]) {
	grid.Fill(0.0)

	if s.DensityWeight == 0 {
		// occupancy^0 is 1 for any occupancy, so every source's seed is 1
		// regardless of local density: skip counting it altogether.
		q := filter.Query()
		for q.NextTable() {
			positions := q.GetColumns()
			for i := range positions {
				pos := &positions[i]
				grid.Set(pos.X, pos.Y, 1.0)
			}
		}
		return
	}

	s.presence.Fill(0.0)

	q := filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			s.presence.Set(pos.X, pos.Y, 1.0)
		}
	}

	s.buildSAT()

	q = filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			occupancy := s.localCount(pos.X, pos.Y) / s.maxCount
			grid.Set(pos.X, pos.Y, math.Pow(occupancy, s.DensityWeight))
		}
	}
}

// buildSAT computes a summed-area table of s.presence into s.sat, so that
// localCount can answer a windowed tree count in O(1) instead of
// O(radius^2) per source.
func (s *TreeAttractionDensityPeak) buildSAT() {
	w, h := s.presence.Width(), s.presence.Height()
	stride := h + 1
	idx := func(x, y int) int { return x*stride + y }

	sat := s.sat
	for x := 0; x <= w; x++ {
		sat[idx(x, 0)] = 0
	}
	for y := 0; y <= h; y++ {
		sat[idx(0, y)] = 0
	}
	for x := 1; x <= w; x++ {
		for y := 1; y <= h; y++ {
			sat[idx(x, y)] = s.presence.Get(x-1, y-1) + sat[idx(x-1, y)] + sat[idx(x, y-1)] - sat[idx(x-1, y-1)]
		}
	}
}

// localCount returns the number of same-type trees within DensityRadius of
// (x, y), inclusive of the tree at (x, y) itself (so the result is always
// >= 1 when called on an actual source cell), via the summed-area table
// built by buildSAT.
func (s *TreeAttractionDensityPeak) localCount(x, y int) float64 {
	w, h := s.presence.Width(), s.presence.Height()
	stride := h + 1
	idx := func(xx, yy int) int { return xx*stride + yy }

	r := s.densityRadiusCells
	x1, x2 := max(x-r, 0), min(x+r, w-1)
	y1, y2 := max(y-r, 0), min(y+r, h-1)

	sat := s.sat
	return sat[idx(x2+1, y2+1)] - sat[idx(x1, y2+1)] - sat[idx(x2+1, y1)] + sat[idx(x1, y1)]
}

// fillGrid turns the seeded values into a smooth attraction field by
// propagating each source's value outward, multiplying by decay per
// orthogonal step and decay^sqrt(2) per diagonal step -- unlike
// TreeAttraction's fillGrid, which subtracts a fixed cost per step and
// floors at 0, multiplicative decay never drives a reachable cell to
// exactly 0, so there is no hard radius. It's still exactly
// TreeAttraction's two-pass max-relaxation otherwise -- see its doc comment
// for why two passes suffice -- and still just as stable and bounded,
// since max of values shrunk by a factor in (0,1) can never exceed the
// largest seed anywhere in the grid (unlike sum, see TreeAttractionSum's
// doc comment for why summing across this same neighbour structure
// diverges instead).
func (s *TreeAttractionDensityPeak) fillGrid(grid *res.Grid[float64]) {
	w, h := grid.Width(), grid.Height()

	// Offsets of the already-updated neighbours seen by a sweep moving in
	// increasing (forward) resp. decreasing (backward) x/y, paired with
	// their step decay factor.
	forward := [4][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}}
	backward := [4][2]int{{1, 1}, {1, 0}, {1, -1}, {0, 1}}
	decays := [4]float64{math.Pow(s.decay, math.Sqrt2), s.decay, math.Pow(s.decay, math.Sqrt2), s.decay}

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
