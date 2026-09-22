package sys

import (
	"math/rand/v2"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[BeetleEmergence]()
}

// BeetleEmergence of beetles from nematode-infected trees.
type BeetleEmergence struct {
	TickOfYear     int     `yaml:"tick_of_year"`
	BeetlesPerTree int     `yaml:"beetles_per_tree"`
	LifeExpectancy float64 `yaml:"life_expectancy"`

	timeRes ecs.Resource[res.Time]
	randRes ecs.Resource[resource.Rand]

	filter  *ecs.Filter1[comp.Position]
	builder *ecs.Map2[comp.BeetlePosition, comp.LifeExpectancy]

	sourceTrees []comp.Position
}

// Initialize the system.
func (s *BeetleEmergence) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)
	s.randRes = s.randRes.New(world)

	s.filter = s.filter.New(world).With(ecs.C[comp.Damaged](), ecs.C[comp.Infected]())
	s.builder = s.builder.New(world)
}

// Update the system.
func (s *BeetleEmergence) Update(_ *ecs.World) {
	time := s.timeRes.Get()

	if time.TickOfYear != s.TickOfYear {
		return
	}
	rng := rand.New(s.randRes.Get())

	q := s.filter.Query()
	for q.NextTable() {
		positions := q.GetColumns()
		s.sourceTrees = append(s.sourceTrees, positions...)
	}

	i := 0
	s.builder.NewBatchFn(len(s.sourceTrees)*s.BeetlesPerTree, func(_ ecs.Entity, bp *comp.BeetlePosition, le *comp.LifeExpectancy) {
		idx := i / s.BeetlesPerTree
		pos := &s.sourceTrees[idx]
		bp.X, bp.Y = pos.X, pos.Y
		le.TickOfDeath = time.Tick + int(rng.ExpFloat64()*s.LifeExpectancy)
		i++
	})

	s.sourceTrees = s.sourceTrees[:0]
}

// Finalize the system.
func (s *BeetleEmergence) Finalize(_ *ecs.World) {}
