package sys

import (
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
)

func init() {
	config.Register[TreeAttraction]()
}

// TreeAttraction system.
type TreeAttraction struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.
	RadiusNear int `yaml:"radius_near"`  // Radius of the near attraction field around each source tree, in meters.
	RadiusFar  int `yaml:"radius_far"`   // Radius of the far attraction field around each source tree, in meters.

	timeRes ecs.Resource[res.Time]

	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	// unitsPerCell is the number of tree-grid units per dispersal-grid
	// cell, i.e. CellSize expressed in the world's base cell-size units.
	unitsPerCell int

	healthyAttractionNear res.Grid[float64]
	damagedAttractionNear res.Grid[float64]
	healthyAttractionFar  res.Grid[float64]
	damagedAttractionFar  res.Grid[float64]
}

// Initialize the system.
func (s *TreeAttraction) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	if s.RadiusNear%ws.CellSize() != 0 {
		panic("Near radius of the dispersal submodel must be a multiple of the world's base cell size.")
	}
	if s.RadiusFar%s.RadiusNear != 0 {
		panic("Far radius of the dispersal submodel must be a multiple of the near radius.")
	}

	s.healthyAttractionNear = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.damagedAttractionNear = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	ecs.AddResource(world, &res.HealthyTreeAttractionNear{Grid: s.healthyAttractionNear})
	ecs.AddResource(world, &res.DamagedTreeAttractionNear{Grid: s.damagedAttractionNear})

	s.unitsPerCell = s.RadiusNear / ws.CellSize()
	width, height := util.CeilDiv(ws.Width(), s.unitsPerCell), util.CeilDiv(ws.Height(), s.unitsPerCell)
	s.healthyAttractionFar = res.NewGrid[float64](width, height, s.RadiusNear)
	s.damagedAttractionFar = res.NewGrid[float64](width, height, s.RadiusNear)
	ecs.AddResource(world, &res.HealthyTreeAttractionFar{Grid: s.healthyAttractionFar})
	ecs.AddResource(world, &res.DamagedTreeAttractionFar{Grid: s.damagedAttractionFar})
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
	s.fillFromQuery(&s.healthyAttractionNear, s.filterHealthy, 1, float64(s.RadiusNear/s.healthyAttractionNear.CellSize()))
	s.fillFromQuery(&s.damagedAttractionNear, s.filterDamaged, 1, float64(s.RadiusNear/s.damagedAttractionNear.CellSize()))

	s.fillGrid(&s.healthyAttractionNear)
	s.fillGrid(&s.damagedAttractionNear)

	s.fillFromQuery(&s.healthyAttractionFar, s.filterHealthy, s.unitsPerCell, float64(s.RadiusFar/s.healthyAttractionFar.CellSize()))
	s.fillFromQuery(&s.damagedAttractionFar, s.filterDamaged, s.unitsPerCell, float64(s.RadiusFar/s.damagedAttractionFar.CellSize()))

	s.fillGrid(&s.healthyAttractionFar)
	s.fillGrid(&s.damagedAttractionFar)
}

// fillFromQuery seeds the attraction grid: every cell containing a matching
// tree is set to the field's peak value (s.Radius), everything else to 0.
// fillGrid then propagates these peaks outward.
func (s *TreeAttraction) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position], unitsPerCell int, peak float64) {
	grid.Fill(0.0)
	q := filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			x, y := s.toCoords(pos.X, pos.Y, unitsPerCell)
			grid.Set(x, y, peak)
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

// toCoords calculates attraction-grid coords from tree-grid coords.
func (s *TreeAttraction) toCoords(x, y, upc int) (int, int) {
	return x / upc, y / upc
}
