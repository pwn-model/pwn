package sys

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
)

func init() {
	config.Register[InitGrids]()
}

// InitGrids system
type InitGrids struct {
	trees res.TreeGrid
}

// Initialize the system.
func (s *InitGrids) Initialize(world *ecs.World) {
	ws := ecs.GetResource[res.WorldSize](world)

	s.trees = res.TreeGrid{
		Grid: res.NewGrid[ecs.Entity](ws.Width(), ws.Height(), ws.CellSize()),
	}
	ecs.AddResource(world, &s.trees)
}

// Update the system.
func (s *InitGrids) Update(_ *ecs.World) {}

// Finalize the system.
func (s *InitGrids) Finalize(_ *ecs.World) {}
