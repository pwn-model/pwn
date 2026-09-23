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
// So this system leaves TreeAttraction's fillGrid completely untouched --
// still a plain max-relaxation distance transform -- and instead makes the
// SEED PEAK fed into it depend on each source's own local tree density
// (see fillFromQuery). A denser cluster starts from a taller peak, so it
// can out-reach and win cells that a closer but sparser source would
// otherwise have claimed by max-relaxation: a real, comparison-relevant
// effect, decided before propagation ever compares sources against each
// other, rather than an inert transform applied after the fact.
//
// Only one of TreeAttraction, TreeAttractionSweep, TreeAttractionRecursive
// or TreeAttractionDensityPeak is meant to be active in a given config,
// since they all publish to the same [res.HealthyTreeAttraction] /
// [res.DamagedTreeAttraction] resources.
type TreeAttractionDensityPeak struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.
	Radius     int `yaml:"radius"`       // Base radius of a single, isolated source tree, in meters.

	// DensityRadius is the radius, in meters, within which a source tree's
	// own same-type neighbours are counted towards its local density.
	// Independent of Radius: this is the "how clustered is this source"
	// scale, not the "how far does its peak reach" scale. Must be a
	// multiple of the world's base cell size.
	DensityRadius int `yaml:"density_radius"`

	// DensityWeight is the exponent applied to a source's local tree count
	// (which always includes the source itself, so it's always >= 1) to
	// scale its peak: 0 recovers plain TreeAttraction (every source peaks
	// at Radius, regardless of clustering), 1 makes peak scale linearly
	// with local tree count, and >1 makes denser clusters reach
	// disproportionately farther than the same trees spread out.
	DensityWeight float64 `yaml:"density_weight"`

	timeRes ecs.Resource[res.Time]

	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	// densityRadiusCells is DensityRadius expressed in grid cells.
	densityRadiusCells int

	healthyAttraction res.Grid[float64]
	damagedAttraction res.Grid[float64]

	// presence is a reused 0/1 scratch grid for whichever type
	// (healthy/damaged) is currently being seeded.
	presence res.Grid[float64]

	// sat is a reused (width+1)*(height+1) summed-area-table buffer for
	// counting each source's local density, laid out the same column-major
	// way as res.Grid with an extra leading zero row/column, as in
	// TreeAttractionSweep's boxBlur.
	sat []float64
}

// Initialize the system.
func (s *TreeAttractionDensityPeak) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	if s.Radius%ws.CellSize() != 0 {
		panic("Radius of the dispersal submodel must be a multiple of the world's base cell size.")
	}
	if s.DensityRadius%ws.CellSize() != 0 {
		panic("DensityRadius of the dispersal submodel must be a multiple of the world's base cell size.")
	}
	s.densityRadiusCells = s.DensityRadius / ws.CellSize()

	s.healthyAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.damagedAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	ecs.AddResource(world, &res.HealthyTreeAttraction{Grid: s.healthyAttraction})
	ecs.AddResource(world, &res.DamagedTreeAttraction{Grid: s.damagedAttraction})

	s.presence = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.sat = make([]float64, (ws.Width()+1)*(ws.Height()+1))
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
	basePeak := float64(s.Radius / s.healthyAttraction.CellSize())

	s.fillFromQuery(&s.healthyAttraction, s.filterHealthy, basePeak)
	s.fillFromQuery(&s.damagedAttraction, s.filterDamaged, basePeak)

	s.fillGrid(&s.healthyAttraction)
	s.fillGrid(&s.damagedAttraction)
}

// fillFromQuery seeds the attraction grid: every cell containing a
// matching tree is set to basePeak scaled by that source's own local
// density (its same-type neighbour count within DensityRadius, raised to
// DensityWeight), everything else to 0. fillGrid then propagates these
// peaks outward exactly as in TreeAttraction.
func (s *TreeAttractionDensityPeak) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position], basePeak float64) {
	grid.Fill(0.0)
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
			count := s.localCount(pos.X, pos.Y)
			grid.Set(pos.X, pos.Y, basePeak*math.Pow(count, s.DensityWeight))
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

// fillGrid turns the seeded peaks into a smooth attraction field by
// propagating each source's value outward, decreasing by 1 per orthogonal
// step and sqrt(2) per diagonal step, floored at 0. Identical to
// TreeAttraction's fillGrid -- see its doc comment for why two passes
// suffice -- since this system's only difference from TreeAttraction is
// how the peaks fed into this function are computed, not how they're
// propagated.
func (s *TreeAttractionDensityPeak) fillGrid(grid *res.Grid[float64]) {
	w, h := grid.Width(), grid.Height()

	// Offsets of the already-updated neighbours seen by a sweep moving in
	// increasing (forward) resp. decreasing (backward) x/y, paired with
	// their step cost.
	forward := [4][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}}
	backward := [4][2]int{{1, 1}, {1, 0}, {1, -1}, {0, 1}}
	costs := [4]float64{math.Sqrt2, 1, math.Sqrt2, 1}

	relax := func(x, y int, offsets [4][2]int) {
		best := grid.Get(x, y)
		for i, o := range offsets {
			nx, ny := x+o[0], y+o[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			if v := grid.Get(nx, ny) - costs[i]; v > best {
				best = v
			}
		}
		if best < 0 {
			best = 0
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
