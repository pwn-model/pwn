package sys

import (
	"fmt"

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
	if ws.Width%ws.Resolution != 0 || ws.Height%ws.Resolution != 0 {
		panic(fmt.Sprintf(
			"world size (%d x %d) must be a multiple of the grid resolution (%d)",
			ws.Width, ws.Height, ws.Resolution,
		))
	}

	s.trees = res.NewEntityGrid(ws.Width, ws.Height)
	ecs.AddResource(world, &s.trees)

	s.grid = res.SpaceGrid{
		EntityGrid: res.NewEntityGrid(ws.Width/ws.Resolution, ws.Height/ws.Resolution),
	}

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
