package sys

import (
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[TreeAttractionSweep]()
}

// sweepPasses is the number of box-blur passes cascaded on top of each
// other. By the central limit theorem, repeatedly box-blurring converges
// towards a Gaussian blur; 3 passes is the usual rule-of-thumb minimum for a
// visually convincing approximation.
const sweepPasses = 3

// TreeAttractionSweep system.
//
// An alternative to [TreeAttraction] that produces a density-sensitive,
// radially symmetric attraction field via repeated box blurring, instead of
// a plain distance-to-nearest-source field. See fillGrid's doc comment for
// why plain nearest-source distance ignores clustering, and boxBlur's for
// why repeated box blurring is used to fix that instead of a single
// separable exponential decay.
//
// Only one of [TreeAttraction], [TreeAttractionSweep] or
// [TreeAttractionRecursive] is meant to be active in a given config, since
// they all publish to the same [res.HealthyTreeAttraction] /
// [res.DamagedTreeAttraction] resources.
type TreeAttractionSweep struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.
	Scale      int `yaml:"scale"`        // Box-blur radius per pass, in meters. Must be a multiple of the world's base cell size.

	// DensityWeight is the exponent applied to the (unbounded-below-by-one,
	// since it's an average of 0/1 presence values) blurred density before
	// it's used as the attraction value: 0 makes attraction density-blind
	// (presence/absence only), 1 makes it linear in local tree density, and
	// >1 makes clusters disproportionately more attractive than the same
	// trees spread out.
	DensityWeight float64 `yaml:"density_weight"`

	timeRes ecs.Resource[res.Time]

	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	// radius is Scale expressed in grid cells.
	radius int

	healthyAttraction res.Grid[float64]
	damagedAttraction res.Grid[float64]

	// sat is a reused (width+1)*(height+1) summed-area-table buffer, laid
	// out the same column-major way as res.Grid, with an extra leading
	// zero row/column so that a box sum never needs a bounds check for its
	// "one before the window" corner.
	sat []float64
}

// Initialize the system.
func (s *TreeAttractionSweep) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	if s.Scale%ws.CellSize() != 0 {
		panic("Scale of the dispersal submodel must be a multiple of the world's base cell size.")
	}
	s.radius = s.Scale / ws.CellSize()

	s.healthyAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.damagedAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	ecs.AddResource(world, &res.HealthyTreeAttraction{Grid: s.healthyAttraction})
	ecs.AddResource(world, &res.DamagedTreeAttraction{Grid: s.damagedAttraction})

	s.sat = make([]float64, (ws.Width()+1)*(ws.Height()+1))
}

// Update the system.
func (s *TreeAttractionSweep) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.calcAttraction()
	}
}

// Finalize the system.
func (s *TreeAttractionSweep) Finalize(_ *ecs.World) {}

func (s *TreeAttractionSweep) calcAttraction() {
	s.fillFromQuery(&s.healthyAttraction, s.filterHealthy)
	s.fillFromQuery(&s.damagedAttraction, s.filterDamaged)

	for range sweepPasses {
		s.boxBlur(&s.healthyAttraction)
		s.boxBlur(&s.damagedAttraction)
	}

	s.applyDensityWeight(&s.healthyAttraction)
	s.applyDensityWeight(&s.damagedAttraction)
}

// fillFromQuery seeds the grid with a 1.0 presence indicator for every
// matching tree, 0 elsewhere. Unlike TreeAttraction's fillFromQuery, there
// is no peak/radius magnitude here: absolute scale doesn't matter for this
// field, only the relative density it ends up encoding.
func (s *TreeAttractionSweep) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position]) {
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

// boxBlur replaces grid in place with the average of each cell's
// (2*radius+1)x(2*radius+1) neighbourhood (zero-padded at the world's
// edges), computed in O(cells) via a summed-area table rather than
// O(cells*radius^2) by brute force.
//
// Called sweepPasses times in calcAttraction: a single box blur is
// square-shaped and not radially symmetric, but by the central limit
// theorem, cascading a few of them converges to an isotropic Gaussian
// blur. That matters here because a plain separable exponential decay
// (`exp(-|dx|/l) * exp(-|dy|/l)`) is diamond-shaped, not circular -- a
// separable kernel is only radially symmetric if its exponent depends on
// dx^2+dy^2 (as a Gaussian's does) rather than |dx|+|dy|.
func (s *TreeAttractionSweep) boxBlur(grid *res.Grid[float64]) {
	w, h := grid.Width(), grid.Height()
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
			sat[idx(x, y)] = grid.Get(x-1, y-1) + sat[idx(x-1, y)] + sat[idx(x, y-1)] - sat[idx(x-1, y-1)]
		}
	}

	r := s.radius
	area := float64((2*r + 1) * (2*r + 1))
	for x := range w {
		x1, x2 := max(x-r, 0), min(x+r, w-1)
		for y := range h {
			y1, y2 := max(y-r, 0), min(y+r, h-1)
			sum := sat[idx(x2+1, y2+1)] - sat[idx(x1, y2+1)] - sat[idx(x2+1, y1)] + sat[idx(x1, y1)]
			grid.Set(x, y, sum/area)
		}
	}
}

// applyDensityWeight raises every cell to the DensityWeight exponent,
// leaving zero cells at zero (0^w is undefined for w<=0, and always 0 for
// w>0 anyway, so it's special-cased rather than left to math.Pow).
func (s *TreeAttractionSweep) applyDensityWeight(grid *res.Grid[float64]) {
	w, h := grid.Width(), grid.Height()
	for x := range w {
		for y := range h {
			d := grid.Get(x, y)
			if d <= 0 {
				grid.Set(x, y, 0)
				continue
			}
			grid.Set(x, y, math.Pow(d, s.DensityWeight))
		}
	}
}
