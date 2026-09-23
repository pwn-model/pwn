package sys

import (
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[TreeAttraction]()
}

// TreeAttraction system.
type TreeAttraction struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.
	Radius     int `yaml:"radius"`       // Radius of the attraction field around each source tree, in meters.

	timeRes ecs.Resource[res.Time]

	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	healthyAttraction res.Grid[float64]
	damagedAttraction res.Grid[float64]
}

// Initialize the system.
func (s *TreeAttraction) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	if s.Radius%ws.CellSize() != 0 {
		panic("Radius of the dispersal submodel must be a multiple of the world's base cell size.")
	}

	s.healthyAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.damagedAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	ecs.AddResource(world, &res.HealthyTreeAttraction{Grid: s.healthyAttraction})
	ecs.AddResource(world, &res.DamagedTreeAttraction{Grid: s.damagedAttraction})
}

// Update the system.
func (s *TreeAttraction) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.calcAttraction()
	}
}

// Finalize the system.
func (s *TreeAttraction) Finalize(_ *ecs.World) {}

func (s *TreeAttraction) calcAttraction() {
	peak := float64(s.Radius / s.healthyAttraction.CellSize())

	s.fillFromQuery(&s.healthyAttraction, s.filterHealthy, peak)
	s.fillFromQuery(&s.damagedAttraction, s.filterDamaged, peak)

	s.fillGrid(&s.healthyAttraction)
	s.fillGrid(&s.damagedAttraction)
}

// fillFromQuery seeds the attraction grid: every cell containing a matching
// tree is set to the field's peak value (s.Radius), everything else to 0.
// fillGrid then propagates these peaks outward.
func (s *TreeAttraction) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position], peak float64) {
	grid.Fill(0.0)
	q := filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			grid.Set(pos.X, pos.Y, peak)
		}
	}
}

// fillGrid turns the seeded peaks into a smooth attraction field by
// propagating each source's value outward, decreasing by 1 per orthogonal
// step and sqrt(2) per diagonal step, floored at 0. This is an in-place,
// single-grid Gauss-Seidel relaxation (a discrete fast-sweeping method):
// two passes in opposite raster directions suffice, because the grid has
// no obstacles, so any shortest path from a source can be split into one
// forward-monotone and one backward-monotone segment, each fully resolved
// by one of the two passes.
func (s *TreeAttraction) fillGrid(grid *res.Grid[float64]) {
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
