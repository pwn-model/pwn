package sys

import (
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
)

// UpdateTime adds and updates the Time resource
type UpdateTime struct {
	WeeksPerYear int

	time    res.Time
	tickRes ecs.Resource[resource.Tick]
}

// Initialize the system.
func (s *UpdateTime) Initialize(world *ecs.World) {
	s.tickRes = s.tickRes.New(world)

	s.time = res.Time{}
	ecs.AddResource(world, &s.time)
}

// Update the system.
func (s *UpdateTime) Update(world *ecs.World) {
	tick := int(s.tickRes.Get().Tick)
	s.time.Tick = tick
	s.time.TickOfYear = tick % s.WeeksPerYear
	s.time.Year = tick / s.WeeksPerYear
}

// Finalize the system.
func (s *UpdateTime) Finalize(_ *ecs.World) {}
