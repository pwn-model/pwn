package sys

import (
	"math"
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
)

func init() {
	config.Register[Colonization]()
}

// Colonization is the background beetle spread process.
type Colonization struct {
	// TickOfYear when colonization takes place.
	TickOfYear int `yaml:"tick_of_year"`
	// CellSize of the dispersal grid, in meters.
	CellSize int `yaml:"cell_size"`
	// KernelHalfDistance is the distance, in meters, at which the dispersal
	// kernel's (pre-normalization) weight has decayed to half its value at
	// the source cell -- i.e. a beetle is half as likely to land this far
	// from a colonized tree as to land in its own cell.
	KernelHalfDistance float64 `yaml:"kernel_half_distance"`
	// KernelRadius is the dispersal kernel's cutoff radius, in meters.
	// Rounded up to full cells.
	KernelRadius int `yaml:"kernel_radius"`
	// BeetlesPerTree is the fixed number of beetles emerging from each
	// colonized tree per year.
	BeetlesPerTree float64 `yaml:"beetles_per_tree"`
	// TreesPerBeetle is the mean number of distinct (uniformly random)
	// damaged trees within a cell that a single arriving beetle attempts
	// to colonize.
	TreesPerBeetle float64 `yaml:"trees_per_beetle"`

	coloFilter    *ecs.Filter1[comp.Position]
	damagedFilter *ecs.Filter1[comp.Position]

	coloMapper *ecs.Map1[comp.Colonized]

	timeRes ecs.Resource[res.Time]
	randRes ecs.Resource[resource.Rand]

	colonizedMap *ecs.Map1[comp.Colonized]

	density     res.Grid[int]
	susceptible res.Grid[int]
	arrivals    res.Grid[float64]
	probability res.Grid[float64]

	kernel []kernelOffset

	toColonize []ecs.Entity

	// unitsPerCell is the number of tree-grid units per dispersal-grid
	// cell, i.e. CellSize expressed in the world's base cell-size units.
	unitsPerCell int
}

// kernelOffset is one pre-computed weighted offset of a dispersal kernel,
// relative to its source cell.
type kernelOffset struct {
	dx, dy int
	weight float64
}

// buildKernel pre-computes a truncated, radially symmetric dispersal kernel
// as a flat list of (offset, weight) pairs, so that per-tick spread only
// needs cheap integer offset lookups instead of a distance calculation
// per cell pair.
//
// Weights are normalized to sum to 1, so the kernel is a proper dispersal
// probability distribution: splatting a fixed number of emerging beetles
// through it conserves beetle count (up to grid-edge losses) instead of
// scaling with the kernel's arbitrary peak.
//
// The negative-exponential decay below is a placeholder for a "simple"
// dispersal kernel; swap the weight formula for whatever shape the model
// needs (Gaussian, power-law, ...) without touching the convolution logic
// in calcArrivals.
func buildKernel(radius int, scale float64) []kernelOffset {
	offsets := make([]kernelOffset, 0, (2*radius+1)*(2*radius+1))
	sum := 0.0
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			d := math.Hypot(float64(dx), float64(dy))
			if d > float64(radius) {
				continue
			}
			w := math.Exp(-d / scale)
			offsets = append(offsets, kernelOffset{dx: dx, dy: dy, weight: w})
			sum += w
		}
	}
	for i := range offsets {
		offsets[i].weight /= sum
	}
	return offsets
}

// Initialize the system.
func (s *Colonization) Initialize(world *ecs.World) {
	s.coloFilter = s.coloFilter.New(world).With(ecs.C[comp.Colonized]())
	s.damagedFilter = s.damagedFilter.New(world).With(ecs.C[comp.Damaged]()).Without(ecs.C[comp.Colonized]())
	s.coloMapper = s.coloMapper.New(world)

	s.timeRes = s.timeRes.New(world)
	s.randRes = s.randRes.New(world)

	s.colonizedMap = s.colonizedMap.New(world)

	ws := ecs.GetResource[res.WorldSize](world)
	if s.CellSize%ws.CellSize() != 0 {
		panic("CellSize of the colonization submodel must be a multiple of the world's base cell size.")
	}
	s.unitsPerCell = s.CellSize / ws.CellSize()

	width, height := util.CeilDiv(ws.Width(), s.unitsPerCell), util.CeilDiv(ws.Height(), s.unitsPerCell)
	s.density = res.NewGrid[int](width, height, s.CellSize)
	s.susceptible = res.NewGrid[int](width, height, s.CellSize)
	s.arrivals = res.NewGrid[float64](width, height, s.CellSize)
	s.probability = res.NewGrid[float64](width, height, s.CellSize)

	s.kernel = buildKernel(util.CeilDiv(s.KernelRadius, s.CellSize), s.KernelHalfDistance/(math.Ln2*float64(s.CellSize)))
}

// Update the system.
func (s *Colonization) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy != s.TickOfYear {
		return
	}

	rng := rand.New(s.randRes.Get())

	s.density.Fill(0)
	s.susceptible.Fill(0)
	s.arrivals.Fill(0.0)
	//s.probability.Fill(0.0)

	// Density of colonized trees.
	q := s.coloFilter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := positions[i]
			x, y := s.toCoords(pos.X, pos.Y)
			v := s.density.Get(x, y)
			s.density.Set(x, y, v+1)
		}
	}

	// The beetles that emerged from a source tree have flown out to lay
	// their eggs elsewhere, so the source tree empties out and becomes
	// available for colonization again.
	s.coloMapper.RemoveBatch(s.coloFilter.Batch(), nil)

	// Density of susceptible trees.
	qd := s.damagedFilter.Query()
	for qd.NextTable() {
		positions := qd.GetColumns()
		for i := range positions {
			pos := positions[i]
			x, y := s.toCoords(pos.X, pos.Y)
			s.susceptible.Set(x, y, s.susceptible.Get(x, y)+1)
		}
	}

	s.calcArrivals()
	s.calcProbability()

	// Colonization
	q = s.damagedFilter.Query()
	for q.NextTable() {
		entities := q.Entities()
		positions := q.GetColumns()
		for i := range positions {
			pos := positions[i]
			x, y := s.toCoords(pos.X, pos.Y)
			p := s.probability.Get(x, y)
			if rng.Float64() < p {
				s.toColonize = append(s.toColonize, entities[i])
			}
		}
	}

	for _, e := range s.toColonize {
		s.colonizedMap.Add(e, &comp.Colonized{})
	}
	s.toColonize = s.toColonize[:0]
}

// calcArrivals applies the pre-normalized dispersal kernel to the density
// grid, computing the deterministic expected number of arriving beetles
// per cell (a fractional value, not a stochastic draw): each colonized
// tree emits a fixed BeetlesPerTree, split across its neighborhood
// according to the kernel's (normalized) weights.
//
// Rather than looping over every target cell and gathering contributions
// from the whole kernel window (O(cells * kernel_area)), this "splats"
// each occupied source cell's contribution onto its neighborhood
// (O(colonized_cells * kernel_area)). That is far cheaper whenever
// colonized cells are a small fraction of the grid, which is the usual
// case for a spreading infestation, and never worse than the gather
// approach.
func (s *Colonization) calcArrivals() {
	w, h := s.density.Width(), s.density.Height()

	for x := range w {
		for y := range h {
			v := s.density.Get(x, y)
			if v == 0 {
				continue
			}
			emitted := float64(v) * s.BeetlesPerTree
			for _, k := range s.kernel {
				nx, ny := x+k.dx, y+k.dy
				if nx < 0 || nx >= w || ny < 0 || ny >= h {
					continue
				}
				arrivals := s.arrivals.Get(nx, ny) + emitted*k.weight
				s.arrivals.Set(nx, ny, arrivals)
			}
		}
	}
}

// calcProbability converts expected beetle arrivals into a per-tree
// colonization probability for each cell, capping colonization by the
// finite supply of arriving beetles competing for the cell's susceptible
// trees, rather than treating each susceptible tree as an independent,
// unbounded Bernoulli trial.
//
// Each arriving beetle is assumed to make TreesPerBeetle independent,
// uniformly random attempts among the damaged trees within its own
// landing cell (approximating multi-tree oviposition without resolving
// which individual trees within the cell are closer together). Attempts
// are not exclusive -- several can land on the same tree, matching the
// lack of any competition/exclusion mechanism between beetles. This is
// the classic occupancy ("balls into bins") problem: the probability
// that a given tree receives at least one attempt out of n independent
// draws among m equally likely trees is 1 - (1 - 1/m)^n. It's computed
// here via log1p/expm1 for numerical stability at large m or n.
func (s *Colonization) calcProbability() {
	w, h := s.arrivals.Width(), s.arrivals.Height()

	for x := range w {
		for y := range h {
			m := s.susceptible.Get(x, y)
			if m == 0 {
				continue
			}
			n := s.arrivals.Get(x, y) * s.TreesPerBeetle
			p := -math.Expm1(n * math.Log1p(-1/float64(m)))
			s.probability.Set(x, y, p)
		}
	}
}

// toCoords calculates dispersal-grid coords from tree grid coords.
func (s *Colonization) toCoords(x, y int) (int, int) {
	return x / s.unitsPerCell, y / s.unitsPerCell
}

// Finalize the system.
func (s *Colonization) Finalize(_ *ecs.World) {}
