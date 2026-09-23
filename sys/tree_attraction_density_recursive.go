package sys

import (
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[TreeAttractionDensityRecursive]()
}

// TreeAttractionDensityRecursive system.
//
// A sixth alternative to [TreeAttraction] (see also [TreeAttractionSweep],
// [TreeAttractionRecursive], [TreeAttractionDensityPeak] and
// [TreeAttractionSum]), combining what worked in two of those for different
// reasons:
//
//   - Like TreeAttractionDensityPeak, each source's own local tree density
//     (see localCount) scales its *seed* value, before anything is
//     aggregated -- not a DensityWeight exponent applied to the finished
//     field, which (see TreeAttractionRecursive/TreeAttractionExponential's
//     design discussion) is a uniform monotonic transform and therefore
//     provably can't change which of two sources wins a deterministic
//     "highest neighbour" comparison. Seeding a cluster's trees with an
//     elevated value, instead, lets that cluster's combined contribution
//     genuinely out-compete a closer but sparser source once summed.
//   - Like TreeAttractionRecursive, propagation is the separable,
//     multiplicative-decay, forward+backward sum per axis, cascaded a few
//     times for isotropy. Unlike TreeAttractionSum's attempt to fold
//     summation into TreeAttraction's *2D* 4-neighbour relaxation (which
//     turned out to diverge: a forward step there gathers from 4
//     neighbours, so it's only stable while 4*decay<1), a single axis only
//     ever gathers from *one* already-updated neighbour per step, so this
//     stays stable for any decay < 1, at any Scale -- no hard radius, and
//     no numerical blow-up.
//
// Only one of TreeAttraction, TreeAttractionSweep, TreeAttractionRecursive,
// TreeAttractionDensityPeak, TreeAttractionSum or
// TreeAttractionDensityRecursive is meant to be active in a given config,
// since they all publish to the same [res.HealthyTreeAttraction] /
// [res.DamagedTreeAttraction] resources.
type TreeAttractionDensityRecursive struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.

	// Scale is the e-folding decay length per smoothing pass, in meters.
	// As in TreeAttractionRecursive, recursivePasses cascaded passes need
	// Scale to be a few multiples of the world's base cell size to
	// converge to an isotropic shape; at Scale close to the cell size, the
	// field stays visibly diamond-shaped.
	Scale int `yaml:"scale"`

	// DensityRadius is the radius, in meters, within which a source tree's
	// own same-type neighbours are counted towards its local density.
	// Independent of Scale: this is the "how clustered is this source"
	// scale, not the "how far does its contribution reach" scale. Must be
	// a multiple of the world's base cell size.
	DensityRadius int `yaml:"density_radius"`

	// DensityWeight is the exponent applied to a source's local tree count
	// (which always includes the source itself, so it's always >= 1) to
	// scale its seed value: 0 recovers a plain (uniformly seeded)
	// TreeAttractionRecursive field, 1 makes the seed scale linearly with
	// local tree count, and >1 makes denser clusters contribute
	// disproportionately more than the same trees spread out.
	DensityWeight float64 `yaml:"density_weight"`

	timeRes ecs.Resource[res.Time]

	filterHealthy *ecs.Filter1[comp.Position]
	filterDamaged *ecs.Filter1[comp.Position]

	// decay is the per-cell-step decay factor derived from Scale.
	decay float64

	// densityRadiusCells is DensityRadius expressed in grid cells.
	densityRadiusCells int

	healthyAttraction res.Grid[float64]
	damagedAttraction res.Grid[float64]

	// presence is a reused 0/1 scratch grid for whichever type
	// (healthy/damaged) is currently being seeded.
	presence res.Grid[float64]

	// sat is a reused (width+1)*(height+1) summed-area-table buffer for
	// counting each source's local density, as in
	// TreeAttractionDensityPeak.
	sat []float64

	// rowOrig/rowFwd/rowBwd are reused scratch buffers, sized to the
	// longer of the grid's two dimensions, so smoothLine never allocates.
	rowOrig []float64
	rowFwd  []float64
	rowBwd  []float64
}

// Initialize the system.
func (s *TreeAttractionDensityRecursive) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	s.filterHealthy = s.filterHealthy.New(world).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	ws := ecs.GetResource[res.WorldSize](world)
	if s.DensityRadius%ws.CellSize() != 0 {
		panic("DensityRadius of the dispersal submodel must be a multiple of the world's base cell size.")
	}
	s.densityRadiusCells = s.DensityRadius / ws.CellSize()
	s.decay = math.Exp(-float64(ws.CellSize()) / float64(s.Scale))

	s.healthyAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.damagedAttraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	ecs.AddResource(world, &res.HealthyTreeAttraction{Grid: s.healthyAttraction})
	ecs.AddResource(world, &res.DamagedTreeAttraction{Grid: s.damagedAttraction})

	s.presence = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.sat = make([]float64, (ws.Width()+1)*(ws.Height()+1))

	n := max(ws.Width(), ws.Height())
	s.rowOrig = make([]float64, n)
	s.rowFwd = make([]float64, n)
	s.rowBwd = make([]float64, n)
}

// Update the system.
func (s *TreeAttractionDensityRecursive) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.calcAttraction()
	}
}

// Finalize the system.
func (s *TreeAttractionDensityRecursive) Finalize(_ *ecs.World) {}

func (s *TreeAttractionDensityRecursive) calcAttraction() {
	s.fillFromQuery(&s.healthyAttraction, s.filterHealthy)
	s.fillFromQuery(&s.damagedAttraction, s.filterDamaged)

	s.smooth(&s.healthyAttraction)
	s.smooth(&s.damagedAttraction)
}

// fillFromQuery seeds the grid with each matching tree's own local density
// (its same-type neighbour count within DensityRadius, raised to
// DensityWeight), 0 elsewhere. smooth then propagates and sums these seeds.
func (s *TreeAttractionDensityRecursive) fillFromQuery(grid *res.Grid[float64], filter *ecs.Filter1[comp.Position]) {
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
			grid.Set(pos.X, pos.Y, math.Pow(count, s.DensityWeight))
		}
	}
}

// buildSAT computes a summed-area table of s.presence into s.sat, so that
// localCount can answer a windowed tree count in O(1) instead of
// O(radius^2) per source.
func (s *TreeAttractionDensityRecursive) buildSAT() {
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
func (s *TreeAttractionDensityRecursive) localCount(x, y int) float64 {
	w, h := s.presence.Width(), s.presence.Height()
	stride := h + 1
	idx := func(xx, yy int) int { return xx*stride + yy }

	r := s.densityRadiusCells
	x1, x2 := max(x-r, 0), min(x+r, w-1)
	y1, y2 := max(y-r, 0), min(y+r, h-1)

	sat := s.sat
	return sat[idx(x2+1, y2+1)] - sat[idx(x1, y2+1)] - sat[idx(x2+1, y1)] + sat[idx(x1, y1)]
}

// smooth turns the seeded grid into an isotropic field by smoothing all
// rows (recursivePasses cascaded passes each), then all columns of the
// result (again recursivePasses cascaded passes each). Identical to
// TreeAttractionRecursive's smooth -- see its doc comment for why cascading
// per axis approximates an isotropic Gaussian rather than a diamond-shaped
// plain exponential decay.
func (s *TreeAttractionDensityRecursive) smooth(grid *res.Grid[float64]) {
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
// the n cells starting at (x0,y0) and stepping by (dx,dy) each time.
// Identical to TreeAttractionRecursive's smoothLine: fwd+bwd-orig gives the
// symmetric kernel result[i] = sum_j orig[j] * decay^|i-j|, summing
// contributions from every seeded cell on the line rather than only the
// nearest one counting.
func (s *TreeAttractionDensityRecursive) smoothLine(grid *res.Grid[float64], x0, y0, dx, dy, n int) {
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
