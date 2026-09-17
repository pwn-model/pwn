package sys

import (
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
)

// DamageTrees is a systems that damages trees, and removes damaged trees.
type DamageTrees struct {
	TickOfYear         int
	DamageProbability  float64
	RemovalProbability float64

	timeRes       ecs.Resource[res.Time]
	randRes       ecs.Resource[resource.Rand]
	gridRes       ecs.Resource[res.EntityGrid]
	filter        *ecs.Filter0
	filterDamaged *ecs.Filter1[comp.Position]

	damageMap *ecs.Map1[comp.Damaged]

	toRemove []comp.Position
	toDamage []ecs.Entity
}

// Initialize the system.
func (s *DamageTrees) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)
	s.randRes = s.randRes.New(world)
	s.gridRes = s.gridRes.New(world)

	s.filter = s.filter.New(world).With(ecs.C[comp.Position]()).Without(ecs.C[comp.Damaged]())
	s.filterDamaged = s.filterDamaged.New(world).With(ecs.C[comp.Damaged]())

	s.damageMap = s.damageMap.New(world)
}

// Update the system.
func (s *DamageTrees) Update(world *ecs.World) {
	toy := s.timeRes.Get().TickOfYear

	if toy != s.TickOfYear {
		return
	}

	rng := rand.New(s.randRes.Get())
	grid := s.gridRes.Get()

	qd := s.filterDamaged.Query()
	for qd.NextTable() {
		positions := qd.GetColumns()
		for i := range positions {
			if rng.Float64() < s.RemovalProbability {
				s.toRemove = append(s.toRemove, positions[i])
			}
		}
	}

	for i := range s.toRemove {
		pos := &s.toRemove[i]
		e := grid.Get(pos.X, pos.Y)

		world.RemoveEntity(e)
		grid.Set(pos.X, pos.Y, ecs.Entity{})
	}
	s.toRemove = s.toRemove[:0]

	q := s.filter.Query()
	for q.NextTable() {
		entities := q.Entities()
		for _, e := range entities {
			if rng.Float64() < s.DamageProbability {
				s.toDamage = append(s.toDamage, e)
			}
		}
	}
	for _, e := range s.toDamage {
		s.damageMap.Add(e, &comp.Damaged{})
	}
	s.toDamage = s.toDamage[:0]
}

// Finalize the system.
func (s *DamageTrees) Finalize(_ *ecs.World) {}
