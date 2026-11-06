package sys

import (
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
//
// Computes one attraction field, meant to bias where dispersing beetles
// fly, for a deterministic consumer that always moves towards whichever
// neighbouring cell has the highest attraction value. It computes healthy-
// or damaged-tree attraction, never both: add it to the scheduler twice,
// once with DamagedTrees false and once true, to get both fields -- each
// instance can use entirely different HalfDistance/DensityRadius/
// DensityWeight values, since healthy and damaged trees plausibly attract
// vectors differently.
//
// Each source tree seeds util.PropagateAttraction with its own local
// occupancy fraction (see fillFromQuery), raised to DensityWeight, instead
// of a uniform value: a denser cluster starts from a taller seed, so it can
// out-reach and win cells that a closer but sparser source would otherwise
// have claimed by max-relaxation.
//
// Along with the field, util.PropagateAttraction traces each cell's
// source: the tree cell its value was propagated from. This is not
// necessarily the nearest tree: a cell won by a farther but denser cluster
// gets a cell of that cluster as its source. With DensityWeight > 0, even a tree's own
// cell can get another source, if a neighbouring dense cluster out-reaches
// its own seed.
type TreeAttraction struct {
	TickOfYear int `yaml:"tick_of_year"` // Tick of year when tree attraction is calculated.

	// DamagedTrees selects which trees this instance computes attraction
	// from and which resource it publishes to: false uses healthy trees and
	// publishes res.HealthyTreeAttraction, true uses damaged trees and
	// publishes res.DamagedTreeAttraction.
	DamagedTrees bool `yaml:"damaged_trees"`

	// HalfDistance is the distance, in meters, at which a source's
	// contribution has decayed to half its value at the source cell --
	// i.e. a beetle is half as attracted to a cell this far from a tree as
	// to the tree's own cell. Internally, decay = exp(-CellSize*ln(2) /
	// HalfDistance), so a cell chamfer_distance(cell, source) meters from a
	// source ends up with decay^chamfer_distance(cell, source) of that
	// source's seed value.
	//
	// HalfDistance also sets the exchange rate between distance and
	// density: a source whose seed is k times another's can out-compete it
	// up to HalfDistance*log2(k) meters farther away (see DensityWeight).
	// A larger HalfDistance both spreads attraction farther and lets
	// density matter over a longer range; a small one makes DensityWeight
	// nearly irrelevant, since almost nothing can outweigh raw proximity.
	HalfDistance float64 `yaml:"half_distance"`

	// DensityRadius is the radius, in meters, within which a source tree's
	// own same-type neighbours are counted towards its local density.
	// Independent of HalfDistance: this is the "how clustered is this
	// source" scale, not the "how far does its seed reach" scale. Must be
	// a multiple of the world's base cell size. Unused, and not validated,
	// when DensityWeight is 0 -- see DensityWeight and util.DensitySeeding.
	DensityRadius int `yaml:"density_radius"`

	// DensityWeight is the exponent applied to a source's local occupancy
	// fraction (its same-type neighbour count within DensityRadius,
	// divided by the number of cells that fit in that radius -- so always
	// in (0, 1]) to scale its seed: 0 makes every source seed at 1
	// regardless of clustering (a plain single-nearest/strongest-source
	// decay field), 1 makes the seed scale linearly with local occupancy,
	// and >1 makes denser clusters reach disproportionately farther than
	// the same trees spread out. Since occupancy is normalized to (0, 1],
	// every seed -- and so, by util.PropagateAttraction's max-relaxation, the
	// whole resulting field -- stays within (0, 1] regardless of
	// DensityRadius or how densely populated the world is.
	DensityWeight float64 `yaml:"density_weight"`

	timeRes ecs.Resource[res.Time]

	filter *ecs.Filter1[comp.Position]

	attraction res.Grid[float64]

	// presence marks the cells containing a matching tree. Recalculated
	// together with the attraction field, and published along with it for
	// other systems (e.g. BeetleMovementByField).
	presence res.Grid[bool]

	// source is the index (see res.Grid.Index) of the tree cell each cell's
	// attraction value was propagated from, res.NoCell where there is none.
	// Recalculated together with the attraction field, and published along
	// with it.
	source res.Grid[uint32]

	// density scales each source's seed by its local density.
	density util.DensitySeeding
}

// Initialize the system.
func (s *TreeAttraction) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

	if s.DamagedTrees {
		s.filter = s.filter.New(world).With(ecs.C[comp.Damaged]())
	} else {
		s.filter = s.filter.New(world).Without(ecs.C[comp.Damaged]())
	}

	ws := ecs.GetResource[res.WorldSize](world)
	s.attraction = res.NewGrid[float64](ws.Width(), ws.Height(), ws.CellSize())
	s.presence = res.NewGrid[bool](ws.Width(), ws.Height(), ws.CellSize())
	if uint64(ws.Width())*uint64(ws.Height()) >= uint64(res.NoCell) {
		panic("world too large for uint32 cell indices of attraction sources")
	}
	s.source = res.NewGrid[uint32](ws.Width(), ws.Height(), ws.CellSize())
	if s.DamagedTrees {
		ecs.AddResource(world, &res.DamagedTreeAttraction{
			Attraction: s.attraction, Presence: s.presence, Source: s.source,
		})
	} else {
		ecs.AddResource(world, &res.HealthyTreeAttraction{
			Attraction: s.attraction, Presence: s.presence, Source: s.source,
		})
	}

	s.density = util.NewDensitySeeding(s.DensityWeight, s.DensityRadius, ws)
}

// Update the system.
func (s *TreeAttraction) Update(_ *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy == s.TickOfYear {
		s.fillFromQuery()
		util.PropagateAttraction(&s.attraction, &s.source, s.HalfDistance)
	}
}

// Finalize the system.
func (s *TreeAttraction) Finalize(_ *ecs.World) {}

// fillFromQuery marks every cell containing a matching tree in s.presence
// and seeds the attraction grid: every such cell is set to that source's own
// local occupancy fraction raised to DensityWeight (see util.DensitySeeding),
// everything else to 0. Every such cell is its own source, every other cell
// has none (res.NoCell). util.PropagateAttraction then propagates these
// seeds and sources outward.
//
// This is the only pass over the trees. It seeds every source at 1, which
// is final for DensityWeight 0 (occupancy^0 is 1 for any occupancy);
// otherwise, util.DensitySeeding.SeedDensity rescales the seeds in a
// single pass over the grid.
func (s *TreeAttraction) fillFromQuery() {
	s.presence.Fill(false)
	s.attraction.Fill(0.0)
	s.source.Fill(res.NoCell)

	q := s.filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		for i := range positions {
			pos := &positions[i]
			s.presence.Set(pos.X, pos.Y, true)
			s.attraction.Set(pos.X, pos.Y, 1.0)
			s.source.Set(pos.X, pos.Y, s.source.Index(pos.X, pos.Y))
		}
	}

	s.density.SeedDensity(&s.presence, &s.attraction)
}
