package sys

import (
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[TreeAttractionSum]()
}

// TreeAttractionSum system.
//
// A fifth alternative to [TreeAttraction] (see also [TreeAttractionSweep],
// [TreeAttractionRecursive] and [TreeAttractionDensityPeak]): the literal
// combinator/decay swap in fillGrid itself -- max becomes sum, and the
// subtractive per-step cost becomes a multiplicative decay -- keeping
// TreeAttraction's exact neighbour offsets and two-pass structure
// otherwise unchanged. See fillGrid's doc comment for the resulting
// stability constraint on Scale.
type TreeAttractionSum struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.
	Scale      int `yaml:"scale"`        // E-folding decay length, in meters.

	timeRes ecs.Resource[res.Time]

	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	// decay is the per-orthogonal-cell-step decay factor derived from
	// Scale; a diagonal step uses decay^sqrt(2), mirroring
	// TreeAttraction's sqrt(2) diagonal cost.
	decay float64

	healthyAttraction res.Grid[float64]
	damagedAttraction res.Grid[float64]
}

// Initialize the system.
func (s *TreeAttractionSum) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	s.decay = math.Exp(-float64(ws.CellSize()) / float64(s.Scale))

	s.healthyAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.damagedAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	ecs.AddResource(world, &res.HealthyTreeAttraction{Grid: s.healthyAttraction})
	ecs.AddResource(world, &res.DamagedTreeAttraction{Grid: s.damagedAttraction})
}

// Update the system.
func (s *TreeAttractionSum) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.calcAttraction()
	}
}

// Finalize the system.
func (s *TreeAttractionSum) Finalize(_ *ecs.World) {}

func (s *TreeAttractionSum) calcAttraction() {
	s.fillFromQuery(&s.healthyAttraction, s.filterHealthy)
	s.fillFromQuery(&s.damagedAttraction, s.filterDamaged)

	s.fillGrid(&s.healthyAttraction)
	s.fillGrid(&s.damagedAttraction)
}

// fillFromQuery seeds the attraction grid with a 1.0 presence indicator for
// every matching tree, 0 elsewhere -- unlike TreeAttraction's fixed
// Radius-derived peak, absolute scale doesn't matter here (see
// TreeAttractionSweep's fillFromQuery for the same reasoning), only the
// relative density the field ends up encoding.
func (s *TreeAttractionSum) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position]) {
	grid.Fill(0.0)
	q := filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			grid.Set(pos.X, pos.Y, 1.0)
		}
	}
}

// fillGrid turns the seeded presence grid into a density-summing
// attraction field, using exactly TreeAttraction's neighbour offsets and
// two-pass forward/backward structure, but with its combinator changed
// from max to sum and its decay changed from subtractive to multiplicative:
// each cell accumulates decay-weighted contributions from its
// already-updated neighbours instead of only keeping the best (nearest)
// one.
//
// This trades TreeAttraction's guaranteed-bounded output for something
// that is only bounded when the decay is weak enough: each forward step
// gathers from 4 neighbours, so the recurrence f = seed + decay*A(f) (A
// summing those 4 neighbours) only converges while decay stays below
// roughly 1/4 -- i.e. while Scale stays close to (or below) the world's
// base cell size. At the larger Scale values that make TreeAttraction's
// other alternatives useful, this instead grows without bound as the grid
// gets bigger, rather than converging to a finite density field; see this
// type's own tests for the concrete numbers.
func (s *TreeAttractionSum) fillGrid(grid *res.Grid[float64]) {
	w, h := grid.Width(), grid.Height()

	forward := [4][2]int{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}}
	backward := [4][2]int{{1, 1}, {1, 0}, {1, -1}, {0, 1}}
	decays := [4]float64{math.Pow(s.decay, math.Sqrt2), s.decay, math.Pow(s.decay, math.Sqrt2), s.decay}

	relax := func(x, y int, offsets [4][2]int) {
		sum := grid.Get(x, y)
		for i, o := range offsets {
			nx, ny := x+o[0], y+o[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			sum += decays[i] * grid.Get(nx, ny)
		}
		grid.Set(x, y, sum)
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
