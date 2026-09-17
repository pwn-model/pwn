package sys

import (
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
)

// DiseaseCourse system.
type DiseaseCourse struct {
	TicksToDamage int

	filter   *ecs.Filter1[comp.Infected]
	mapper   *ecs.Map1[comp.Damaged]
	ticksRes ecs.Resource[resource.Tick]

	toDamage []ecs.Entity
}

// Initialize the system.
func (s *DiseaseCourse) Initialize(world *ecs.World) {
	s.filter = s.filter.New(world).Without(ecs.C[comp.Damaged]())
	s.mapper = s.mapper.New(world)
	s.ticksRes = s.ticksRes.New(world)
}

// Update the system.
func (s *DiseaseCourse) Update(_ *ecs.World) {
	tick := s.ticksRes.Get().Tick
	ticksToDamage := int64(s.TicksToDamage)

	query := s.filter.Query()
	for query.NextTable() {
		entities := query.Entities()
		infected := query.GetColumns()
		for i := range infected {
			inf := &infected[i]
			if tick >= inf.InfectionTick+ticksToDamage {
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
