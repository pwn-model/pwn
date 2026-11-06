package sys

import (
	"fmt"
	"math"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
)

func init() {
	config.Register[TreeAttractionOctants]()
}

// TreeAttractionOctants system.
//
// Computes, for every cell, the most attractive tree in each of the 8
// half-open octants around it (see [res.OctantSources]), meant for a
// consumer that selects an octant by the attraction values of its sources
// and moves directly to the selected octant's source tree. A cell's own
// tree is never one of its sources, so that a beetle on a tree "sees" only
// other trees.
//
// Attraction is defined just like for [TreeAttraction]: each tree seeds
// with its local occupancy fraction raised to DensityWeight (see
// [util.DensitySeeding]), decayed by half every HalfDistance meters of
// chamfer distance. Unlike TreeAttraction's single max-relaxed field, the
// octant sources are never shadowed by nearer but less attractive trees in
// the same octant (see [util.PropagateOctants]).
//
// It computes healthy- or damaged-tree octants, never both: add it to the
// scheduler twice, once with DamagedTrees false and once true, to get both,
// each with its own parameters.
//
// Mirrors the sibling Julia implementation's TreeAttractionOctants.
type TreeAttractionOctants struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when the octant sources are calculated.

	// DamagedTrees selects which trees this instance computes octant
	// sources from and which resource it publishes to: false uses healthy
	// trees and publishes res.HealthyTreeAttractionOctants, true uses
	// damaged trees and publishes res.DamagedTreeAttractionOctants.
	DamagedTrees bool `yaml:"damaged_trees"`

	// HalfDistance is the distance, in meters, at which a tree's attraction
	// has decayed to half its seed value. It also sets the exchange rate
	// between distance and density: a tree whose seed is k times another's
	// is equally attractive HalfDistance*log2(k) meters farther away. See
	// TreeAttraction.
	HalfDistance float64 `yaml:"half_distance"`

	// DensityRadius is the radius, in meters, within which a tree's own
	// same-type neighbours are counted towards its local density. Must be a
	// multiple of the world's base cell size. Unused, and not validated,
	// when DensityWeight is 0. See TreeAttraction.
	DensityRadius int `yaml:"density_radius"`

	// DensityWeight is the exponent applied to a tree's local occupancy
	// fraction to get its seed, in (0, 1]: 0 seeds every tree at 1, 1 makes
	// the seed scale linearly with local occupancy. See TreeAttraction.
	DensityWeight float64 `yaml:"density_weight"`

	timeRes ecs.Resource[res.Time]

	filter *ecs.Filter1[comp.Position]

	// seeds is the reused seed grid passed to util.PropagateOctants.
	seeds res.Grid[float64]

	// presence marks the cells containing a matching tree. Recalculated
	// together with the octant sources, and published along with them.
	presence res.Grid[bool]

	// sources holds the octant sources of each cell. Published.
	sources res.Grid[res.OctantSources]

	// buffers are the reused scratch columns of util.PropagateOctants.
	buffers util.OctantBuffers

	// density scales each tree's seed by its local density.
	density util.DensitySeeding
}

// Initialize the system.
func (s *TreeAttractionOctants) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	if s.DamagedTrees {
		s.filter = s.filter.New(world).With(ecs.C[comp.Damaged]())
	} else {
		s.filter = s.filter.New(world).Without(ecs.C[comp.Damaged]())
	}

	ws := ecs.GetResource[res.WorldSize](world)

	// Offsets to sources range from -(size-1) to size-1 cells per axis, and
	// are stored as int16.
	maxSize := math.MaxInt16 + 1
	if ws.Width() > maxSize || ws.Height() > maxSize {
		panic(fmt.Sprintf("world too large for int16 offsets of octant sources, max. %d cells per side", maxSize))
	}

	s.seeds = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.presence = res.NewGrid[bool](ws.Width(), ws.Height(), ws.CellSize())
	s.sources = res.NewGrid[res.OctantSources](ws.Width(), ws.Height(), ws.CellSize())
	s.buffers = util.NewOctantBuffers(ws.Height())
	s.density = util.NewDensitySeeding(s.DensityWeight, s.DensityRadius, ws)

	if s.DamagedTrees {
		ecs.AddResource(world, &res.DamagedTreeAttractionOctants{Sources: s.sources, Presence: s.presence})
	} else {
		ecs.AddResource(world, &res.HealthyTreeAttractionOctants{Sources: s.sources, Presence: s.presence})
	}
}

// Update the system.
func (s *TreeAttractionOctants) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.fillFromQuery()
		util.PropagateOctants(&s.sources, &s.seeds, s.HalfDistance, &s.buffers)
	}
}

// Finalize the system.
func (s *TreeAttractionOctants) Finalize(_ *ecs.World) {}

// fillFromQuery marks every cell containing a matching tree in s.presence
// and sets its seed in s.seeds to the tree's local occupancy fraction
// raised to DensityWeight (see util.DensitySeeding), everything else to 0.
func (s *TreeAttractionOctants) fillFromQuery() {
	s.presence.Fill(false)
	s.seeds.Fill(0.0)

	q := s.filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			s.presence.Set(pos.X, pos.Y, true)
			s.seeds.Set(pos.X, pos.Y, 1.0)
		}
	}

	s.density.SeedDensity(&s.presence, &s.seeds)
}
