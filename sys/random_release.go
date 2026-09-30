package sys

import (
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
)

func init() {
	config.Register[RandomRelease]()
}

// RandomRelease infects the given number of trees in the given grid cell.
type RandomRelease struct {
	// TickOfInfection is the model tick at which trees are infected.
	TickOfInfection int `yaml:"tick_of_infection"`
	// NumTrees is the number of trees to infect. Capped at the number of
	// undamaged, uninfected trees in the cell.
	NumTrees int `yaml:"num_trees"`
	// CellX and CellY are the 0-based column and row of the release cell
	// in the coarse res.SpaceGrid (cells of WorldSize's grid_cell_size
	// meters), counted from the world origin. E.g. with grid_cell_size 500,
	// cell (2, 1) covers x in [1000, 1500) and y in [500, 1000) meters.
	//
	// The sibling Julia implementation uses the same 0-based convention,
	// so that the same config file selects the same cell in both.
	CellX int `yaml:"cell_x"`
	CellY int `yaml:"cell_y"`

	filter  *ecs.Filter2[comp.Position, comp.InCell]
	mapper  *ecs.Map1[comp.Infected]
	timeRes ecs.Resource[res.Time]
}

// Initialize the system.
func (s *RandomRelease) Initialize(world *ecs.World) {
	s.filter = s.filter.New(world).Without(ecs.C[comp.Damaged](), ecs.C[comp.Infected]())
	s.mapper = s.mapper.New(world)
	s.timeRes = s.timeRes.New(world)
}

// Update the system.
func (s *RandomRelease) Update(world *ecs.World) {
	tick := s.timeRes.Get().Tick

	if tick != s.TickOfInfection {
		return
	}

	rng := ecs.GetResource[resource.Rand](world)
	worldSize := ecs.GetResource[res.WorldSize](world)
	grid := ecs.GetResource[res.SpaceGrid](world)
	cell := grid.Get(s.CellX, s.CellY)

	toInfect := make([]ecs.Entity, 0, worldSize.Resolution()*worldSize.Resolution())

	query := s.filter.Query(ecs.RelIdx(1, cell))
	for query.NextTable() {
		toInfect = append(toInfect, query.Entities()...)
	}

	util.Shuffle(rng, toInfect)
	for i := range min(len(toInfect), s.NumTrees) {
		s.mapper.Add(toInfect[i], &comp.Infected{InfectionTick: tick})
	}
}

// Finalize the system.
func (s *RandomRelease) Finalize(_ *ecs.World) {}
