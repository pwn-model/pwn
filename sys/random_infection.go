package sys

import (
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
)

// RandomInfection infects the given number of trees in the given grid cell.
type RandomInfection struct {
	TickOfInfection int
	NumTrees        int
	CellX, CellY    int

	filter  *ecs.Filter2[comp.Position, comp.InCell]
	mapper  *ecs.Map1[comp.Infected]
	timeRes ecs.Resource[res.Time]
}

// Initialize the system.
func (s *RandomInfection) Initialize(world *ecs.World) {
	s.filter = s.filter.New(world).Without(ecs.C[comp.Damaged](), ecs.C[comp.Infected]())
	s.mapper = s.mapper.New(world)
	s.timeRes = s.timeRes.New(world)
}

// Update the system.
func (s *RandomInfection) Update(world *ecs.World) {
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
func (s *RandomInfection) Finalize(_ *ecs.World) {}
