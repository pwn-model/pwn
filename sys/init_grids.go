package sys

import (
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
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
	s.grid = res.SpaceGrid{EntityGrid: res.NewEntityGrid(gridWidth, gridHeight)}

	builder := ecs.NewMap1[comp.GridCoords](world)
	x, y := 0, 0
	builder.NewBatchFn(s.grid.Width()*s.grid.Height(), func(e ecs.Entity, gc *comp.GridCoords) {
		gc.X = x
		gc.Y = y
		s.grid.Set(x, y, e)
		y++
		if y >= s.grid.Height() {
			y = 0
			x++
		}
	})

	ecs.AddResource(world, &s.grid)
}

// Update the system.
func (s *InitGrids) Update(_ *ecs.World) {}

// Finalize the system.
func (s *InitGrids) Finalize(_ *ecs.World) {}
