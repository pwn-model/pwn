package sys

import (
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[TreeAttractionRecursive]()
}

// recursivePasses is the number of single-pole exponential smoothing
// passes cascaded along each axis. By the central limit theorem, cascading
// a few of these converges towards a Gaussian response along that axis.
const recursivePasses = 3

// TreeAttractionRecursive system.
//
// Another alternative to [TreeAttraction] (see also [TreeAttractionSweep]),
// producing a density-sensitive, radially symmetric attraction field. Where
// TreeAttractionSweep reaches for that via a summed-area table (a scan
// building a cumulative sum, then O(1) windowed lookups), this system stays
// structurally close to TreeAttraction's original fast-sweep: a forward
// pass and a backward pass per axis, each cell combining with its
// already-visited neighbour -- just combining by decayed summation instead
// of by max, and cascaded a few times per axis for an isotropic result (see
// smooth's doc comment).
//
// Only one of [TreeAttraction], [TreeAttractionSweep] or
// TreeAttractionRecursive is meant to be active in a given config, since
// they all publish to the same [res.HealthyTreeAttraction] /
// [res.DamagedTreeAttraction] resources.
type TreeAttractionRecursive struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.

	// Scale is the e-folding decay length per smoothing pass, in meters.
	// recursivePasses cascaded passes need Scale to be a few multiples of
	// the world's base cell size to converge to an isotropic shape (see
	// smooth's doc comment); at Scale close to the cell size, the field
	// stays visibly diamond-shaped.
	Scale int `yaml:"scale"`

	// DensityWeight is the exponent applied to the smoothed (unbounded, ---
	// unlike TreeAttractionSweep's average --- a sum of decayed
	// contributions) density before it's used as the attraction value: 0
	// makes attraction density-blind (presence/absence only), 1 makes it
	// linear in local tree density, and >1 makes clusters
	// disproportionately more attractive than the same trees spread out.
	DensityWeight float64 `yaml:"density_weight"`

	timeRes ecs.Resource[res.Time]

	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	// decay is the per-cell-step decay factor derived from Scale.
	decay float64

	healthyAttraction res.Grid[float64]
	damagedAttraction res.Grid[float64]

	// rowOrig/rowFwd/rowBwd are reused scratch buffers, sized to the
	// longer of the grid's two dimensions, so smoothLine never allocates.
	rowOrig []float64
	rowFwd  []float64
	rowBwd  []float64
}

// Initialize the system.
func (s *TreeAttractionRecursive) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	s.decay = math.Exp(-float64(ws.CellSize()) / float64(s.Scale))

	s.healthyAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.damagedAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	ecs.AddResource(world, &res.HealthyTreeAttraction{Grid: s.healthyAttraction})
	ecs.AddResource(world, &res.DamagedTreeAttraction{Grid: s.damagedAttraction})

	n := max(ws.Width(), ws.Height())
	s.rowOrig = make([]float64, n)
	s.rowFwd = make([]float64, n)
	s.rowBwd = make([]float64, n)
}

// Update the system.
func (s *TreeAttractionRecursive) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.calcAttraction()
	}
}

// Finalize the system.
func (s *TreeAttractionRecursive) Finalize(_ *ecs.World) {}

func (s *TreeAttractionRecursive) calcAttraction() {
	s.fillFromQuery(&s.healthyAttraction, s.filterHealthy)
	s.fillFromQuery(&s.damagedAttraction, s.filterDamaged)

	s.smooth(&s.healthyAttraction)
	s.smooth(&s.damagedAttraction)

	s.applyDensityWeight(&s.healthyAttraction)
	s.applyDensityWeight(&s.damagedAttraction)
}

// fillFromQuery seeds the grid with a 1.0 presence indicator for every
// matching tree, 0 elsewhere. As in TreeAttractionSweep, there is no
// peak/radius magnitude: absolute scale doesn't matter here, only the
// relative density the field ends up encoding.
func (s *TreeAttractionRecursive) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position]) {
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

// smooth turns the seeded presence grid into an isotropic, density-weighted
// field by smoothing all rows (recursivePasses cascaded passes each), then
// all columns of the result (again recursivePasses cascaded passes each).
//
// A *single* forward+backward exponential smoothing pass per axis would
// combine into a plain separable exponential decay -- diamond-shaped, not
// circular, since `decay^|dx| * decay^|dy| = decay^(|dx|+|dy|)` depends on
// the L1 (Manhattan) distance. Cascading several such passes along the same
// axis first, by the central limit theorem, rounds that axis's response
// towards a Gaussian; doing that on both axes before combining them means
// the final product is a close approximation of an isotropic 2D Gaussian,
// since a Gaussian's exponent depends on dx^2+dy^2 (Euclidean distance
// squared) and is the only shape both separable and radially symmetric.
func (s *TreeAttractionRecursive) smooth(grid *res.Grid[float64]) {
	w, h := grid.Width(), grid.Height()

	for y := range h {
		for range recursivePasses {
			s.smoothLine(grid, 0, y, 1, 0, w)
		}
	}
	for x := range w {
		for range recursivePasses {
			s.smoothLine(grid, x, 0, 0, 1, h)
		}
	}
}

// smoothLine applies one forward+backward exponential smoothing pass to
// the n cells starting at (x0,y0) and stepping by (dx,dy) each time (a row
// when (dx,dy) is (1,0), a column when it's (0,1)).
//
// The forward pass is a causal recursive filter (fwd[i] depends on
// fwd[i-1]) and the backward pass is its anti-causal mirror; combined as
// fwd+bwd-orig, they give the symmetric kernel
// result[i] = sum_j orig[j] * decay^|i-j|, i.e. each cell ends up as a
// decay-weighted sum of every cell on the line -- unlike TreeAttraction's
// max-relaxation, contributions from multiple nearby sources add up rather
// than only the closest one counting.
func (s *TreeAttractionRecursive) smoothLine(grid *res.Grid[float64], x0, y0, dx, dy, n int) {
	orig, fwd, bwd := s.rowOrig[:n], s.rowFwd[:n], s.rowBwd[:n]

	for i := range n {
		orig[i] = grid.Get(x0+i*dx, y0+i*dy)
	}

	prev := 0.0
	for i := range n {
		v := orig[i] + s.decay*prev
		fwd[i] = v
		prev = v
	}

	prev = 0.0
	for i := n - 1; i >= 0; i-- {
		v := orig[i] + s.decay*prev
		bwd[i] = v
		prev = v
	}

	for i := range n {
		grid.Set(x0+i*dx, y0+i*dy, fwd[i]+bwd[i]-orig[i])
	}
}

// applyDensityWeight raises every cell to the DensityWeight exponent,
// leaving zero (and, since this field is a sum rather than an average,
// negative-epsilon-from-rounding) cells at zero.
func (s *TreeAttractionRecursive) applyDensityWeight(grid *res.Grid[float64]) {
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
