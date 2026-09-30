package sys

import (
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/config"
)

func init() {
	config.Register[FixedTermination]()
}

// FixedTermination terminates a run after a fixed number of ticks.
//
// Ported from ark-tools' system.FixedTermination, so that it is owned (and
// config-named) the same way as all other systems of this model, and its
// parameter is config-decodable by the same snake_case convention.
type FixedTermination struct {
	Steps int `yaml:"steps"` // Number of simulation ticks to run.

	tickRes ecs.Resource[resource.Tick]
	termRes ecs.Resource[resource.Termination]
}

// Initialize the system.
func (s *FixedTermination) Initialize(world *ecs.World) {
	s.tickRes = s.tickRes.New(world)
	s.termRes = s.termRes.New(world)
}

// Update the system.
func (s *FixedTermination) Update(_ *ecs.World) {
	tick := s.tickRes.Get().Tick

	if tick+1 >= int64(s.Steps) {
		s.termRes.Get().Terminate = true
	}
}

// Finalize the system.
func (s *FixedTermination) Finalize(_ *ecs.World) {}
