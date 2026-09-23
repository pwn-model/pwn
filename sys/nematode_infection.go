package sys

import (
	"math"
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[NematodeInfection]()
}

// NematodeInfection from feeding, infected beetles.
type NematodeInfection struct {
	InfectionProbability float64 `yaml:"infection_probability"`

	timeRes    ecs.Resource[res.Time]
	feedingRes ecs.Resource[res.FeedingInfectedBeetles]
	randRes    ecs.Resource[resource.Rand]

	filter *ecs.Filter1[comp.Position]
	mapper *ecs.Map1[comp.Infected]

	toInfect []ecs.Entity
}

// Initialize the system.
func (s *NematodeInfection) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)
	s.feedingRes = s.feedingRes.New(world)
	s.randRes = s.randRes.New(world)
	s.filter = s.filter.New(world).Without(ecs.C[comp.Damaged](), ecs.C[comp.Infected]())
	s.mapper = s.mapper.New(world)
}

// Update the system.
func (s *NematodeInfection) Update(world *ecs.World) {
	tick := s.timeRes.Get().Tick
	feeding := s.feedingRes.Get().Grid
	rng := rand.New(s.randRes.Get())
	nonInfProb := 1.0 - s.InfectionProbability

	q := s.filter.Query()
	for q.NextTable() {
		entities := q.Entities()
		positions := q.GetColumns()
		for i := range entities {
			pos := &positions[i]
			f := feeding.Get(pos.X, pos.Y)
			if f == 0 {
				continue
			}
			p := 1.0 - math.Pow(nonInfProb, float64(f))
			if rng.Float64() < p {
				s.toInfect = append(s.toInfect, entities[i])
			}
		}
	}

	for _, e := range s.toInfect {
		s.mapper.Add(e, &comp.Infected{InfectionTick: tick})
	}
	s.toInfect = s.toInfect[:0]
}

// Finalize the system.
func (s *NematodeInfection) Finalize(_ *ecs.World) {}
