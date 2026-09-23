package sys

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[BeetleMortality]()
}

// BeetleMortality system.
type BeetleMortality struct {
	timeRes ecs.Resource[res.Time]

	filter   *ecs.Filter1[comp.LifeExpectancy]
	toRemove []ecs.Entity
}

// Initialize the system.
func (s *BeetleMortality) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)
	s.filter = s.filter.New(world)
}

// Update the system.
func (s *BeetleMortality) Update(world *ecs.World) {
	tick := s.timeRes.Get().Tick

	q := s.filter.Query()
	for q.NextTable() {
		entities := q.Entities()
		lifeExpectancies := q.GetColumns()
		for i := range lifeExpectancies {
			le := &lifeExpectancies[i]
			if tick >= le.TickOfDeath {
				s.toRemove = append(s.toRemove, entities[i])
			}
		}

	}

	for _, e := range s.toRemove {
		world.RemoveEntity(e)
	}

	s.toRemove = s.toRemove[:0]
}

// Finalize the system.
func (s *BeetleMortality) Finalize(_ *ecs.World) {}
