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

	filter  *ecs.Filter1[comp.Position]
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

	toInfect := make([]ecs.Entity, 0, worldSize.Width()*worldSize.Height())

	query := s.filter.Query()
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
