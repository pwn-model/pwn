package sys

import (
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
)

// RandomInfection infects the given number of trees in the given grid cell.
type RandomInfection struct {
	TickOfInfection int
	NumTrees        int
	CellX, CellY    int

	filter  *ecs.Filter2[comp.Position, comp.InCell]
	mapper  *ecs.Map1[comp.NematodeInfected]
	tickRes ecs.Resource[resource.Tick]
}

// Initialize the system.
func (s *RandomInfection) Initialize(world *ecs.World) {
	s.filter = s.filter.New(world).Without(ecs.C[comp.Damaged](), ecs.C[comp.NematodeInfected]())
	s.mapper = s.mapper.New(world)
	s.tickRes = s.tickRes.New(world)
}

// Update the system.
func (s *RandomInfection) Update(world *ecs.World) {
	tick := s.tickRes.Get().Tick

	if tick != int64(s.TickOfInfection) {
		return
	}

	rng := rand.New(ecs.GetResource[resource.Rand](world))
	worldSize := ecs.GetResource[res.WorldSize](world)
	grid := ecs.GetResource[res.SpaceGrid](world)
	cell := grid.Get(s.CellX, s.CellY)

	toInfect := make([]ecs.Entity, 0, worldSize.Resolution*worldSize.Resolution)

	query := s.filter.Query(ecs.RelIdx(1, cell))
	for query.NextTable() {
		toInfect = append(toInfect, query.Entities()...)
	}

	rng.Shuffle(len(toInfect), func(i, j int) { toInfect[i], toInfect[j] = toInfect[j], toInfect[i] })
	for i := range min(len(toInfect), s.NumTrees) {
		s.mapper.Add(toInfect[i], &comp.NematodeInfected{InfectionTick: tick})
	}
}

// Finalize the system.
func (s *RandomInfection) Finalize(_ *ecs.World) {}
