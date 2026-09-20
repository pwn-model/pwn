package sys

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[DiseaseCourse]()
}

// DiseaseCourse system.
type DiseaseCourse struct {
	TicksToDamage int `yaml:"ticks_to_damage"`

	filter  *ecs.Filter1[comp.Infected]
	mapper  *ecs.Map1[comp.Damaged]
	timeRes ecs.Resource[res.Time]

	toDamage []ecs.Entity
}

// Initialize the system.
func (s *DiseaseCourse) Initialize(world *ecs.World) {
	s.filter = s.filter.New(world).Without(ecs.C[comp.Damaged]())
	s.mapper = s.mapper.New(world)
	s.timeRes = s.timeRes.New(world)
}

// Update the system.
func (s *DiseaseCourse) Update(_ *ecs.World) {
	tick := s.timeRes.Get().Tick

	query := s.filter.Query()
	for query.NextTable() {
		entities := query.Entities()
		infected := query.GetColumns()
		for i := range infected {
			inf := &infected[i]
			if tick >= inf.InfectionTick+s.TicksToDamage {
				s.toDamage = append(s.toDamage, entities[i])
			}
		}
	}

	for _, e := range s.toDamage {
		s.mapper.Add(e, &comp.Damaged{})
	}

	s.toDamage = s.toDamage[:0]
}

// Finalize the system.
func (s *DiseaseCourse) Finalize(_ *ecs.World) {}
