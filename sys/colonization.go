package sys

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
)

// Colonization is the background beetle spread process.
type Colonization struct {
	timeRes ecs.Resource[res.Time]
}

// Initialize the system.
func (s *Colonization) Initialize(world *ecs.World) {
	s.timeRes = s.timeRes.New(world)

}

// Update the system.
func (s *Colonization) Update(_ *ecs.World) {
}

// Finalize the system.
func (s *Colonization) Finalize(_ *ecs.World) {}
