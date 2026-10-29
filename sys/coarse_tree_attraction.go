package sys

import (
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[CoarseTreeAttraction]()
}

// CoarseTreeAttraction system.
//
// Computes one attraction field like [TreeAttraction], but on a coarse grid
// of CellSize meters instead of the world's base tree grid. It computes
// healthy- or damaged-tree attraction, never both: add it to the scheduler
// twice, once with DamagedTrees false and once true, to get both fields.
//
// Instead of TreeAttraction's per-source sliding-window density, each coarse
// cell is seeded with its own occupancy fraction -- the number of matching
// trees in it, divided by the number of base grid cells in a coarse cell --
// raised to DensityWeight. Cells without a matching tree are seeded with 0.
// The seeds are then propagated outward by propagateAttraction, exactly as
// in TreeAttraction.
//
// If the world size is not a multiple of CellSize, the coarse grid is
// rounded up. The partial cells along the right/bottom edge are normalized
// like all others, so they can't reach an occupancy of 1 -- just as
// TreeAttraction normalizes by its full density window, even where that
// window is clipped by the world's edge.
type CoarseTreeAttraction struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.

	// DamagedTrees selects which trees this instance computes attraction
	// from and which resource it publishes to: false uses healthy trees and
	// publishes res.HealthyTreeAttractionCoarse, true uses damaged trees and
	// publishes res.DamagedTreeAttractionCoarse.
	DamagedTrees bool `yaml:"damaged_trees"`

	// CellSize is the coarse grid's cell size, in meters. Must be a multiple
	// of the world's base cell size.
	CellSize int `yaml:"cell_size"`

	// HalfDistance is the distance, in meters, at which a source's
	// contribution has decayed to half its value at the source cell. See
	// TreeAttraction.HalfDistance; here, a step is one coarse cell.
	HalfDistance float64 `yaml:"half_distance"`

	// DensityWeight is the exponent applied to a coarse cell's occupancy
	// fraction (always in (0, 1] for cells with at least one matching tree)
	// to get its seed: 0 seeds every occupied cell at 1, 1 makes the seed
	// scale linearly with occupancy, and >1 favours dense cells
	// disproportionately. As with TreeAttraction, the whole resulting field
	// stays within [0, 1].
	DensityWeight float64 `yaml:"density_weight"`

	timeRes ecs.Resource[res.Time]

	filter *ecs.Filter1[comp.Position]

	// factor is the number of base grid cells per coarse cell along each
	// axis, i.e. CellSize divided by the world's base cell size.
	factor int

	attraction res.Grid[float64]
}

// Initialize the system.
func (s *CoarseTreeAttraction) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	if s.DamagedTrees {
		s.filter = s.filter.New(world).With(ecs.C[comp.Damaged]())
	} else {
		s.filter = s.filter.New(world).Without(ecs.C[comp.Damaged]())
	}

	ws := ecs.GetResource[res.WorldSize](world)
	if s.CellSize <= 0 || s.CellSize%ws.CellSize() != 0 {
		panic("CellSize of CoarseTreeAttraction must be a positive multiple of the world's base cell size.")
	}
	s.factor = s.CellSize / ws.CellSize()

	w := (ws.Width() + s.factor - 1) / s.factor
	h := (ws.Height() + s.factor - 1) / s.factor
	s.attraction = res.NewGrid[float64](w, h, s.CellSize)
	if s.DamagedTrees {
		ecs.AddResource(world, &res.DamagedTreeAttractionCoarse{Grid: s.attraction})
	} else {
		ecs.AddResource(world, &res.HealthyTreeAttractionCoarse{Grid: s.attraction})
	}
}

// Update the system.
func (s *CoarseTreeAttraction) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.calcAttraction()
	}
}

// Finalize the system.
func (s *CoarseTreeAttraction) Finalize(_ *ecs.World) {}

func (s *CoarseTreeAttraction) calcAttraction() {
	s.fillFromQuery(&s.attraction, s.filter)
	propagateAttraction(&s.attraction, s.HalfDistance)
}

// fillFromQuery seeds the coarse attraction grid: every coarse cell is set
// to its occupancy fraction (matching tree count divided by maxCount)
// raised to DensityWeight, or to 0 if it holds no
// matching tree. propagateAttraction then propagates these seeds outward.
func (s *CoarseTreeAttraction) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position]) {
	grid.Fill(0.0)

	f := s.factor
	maxCount := float64(f * f)

	q := filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			cx, cy := pos.X/f, pos.Y/f
			grid.Set(cx, cy, grid.Get(cx, cy)+1)
		}
	}

	w, h := grid.Width(), grid.Height()
	for x := range w {
		for y := range h {
			count := grid.Get(x, y)
			if count == 0 {
				continue
			}
			grid.Set(x, y, math.Pow(count/maxCount, s.DensityWeight))
		}
	}
}
