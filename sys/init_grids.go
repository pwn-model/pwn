package sys

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
)

// InitGrids system
type InitGrids struct {
	trees res.EntityGrid
	grid  res.SpaceGrid
}

// Initialize the system.
func (s *InitGrids) Initialize(world *ecs.World) {
	ws := ecs.GetResource[res.WorldSize](world)

	s.trees = res.NewEntityGrid(ws.Width, ws.Height)
	ecs.AddResource(world, &s.trees)

	gridHeight := (ws.Height + ws.Resolution - 1) / ws.Resolution
	gridWidth := (ws.Width + ws.Resolution - 1) / ws.Resolution
	s.grid = (res.SpaceGrid)(res.NewEntityGrid(gridWidth, gridHeight))
	ecs.AddResource(world, &s.grid)
}

// Update the system.
func (s *InitGrids) Update(_ *ecs.World) {}

// Finalize the system.
func (s *InitGrids) Finalize(_ *ecs.World) {}
