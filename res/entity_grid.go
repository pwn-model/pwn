package res

import "github.com/mlange-42/ark/ecs"

// EntityGrid resource.
//
// Entities are stored column-major (consecutive y for fixed x are contiguous),
// so that column-wise iteration is cache-friendly for large grids.
type EntityGrid struct {
	entities []ecs.Entity
	width    int
	height   int
}

// NewEntityGrid creates a new [EntityGrid] of the given size.
func NewEntityGrid(sx, sy int) EntityGrid {
	return EntityGrid{
		entities: make([]ecs.Entity, sx*sy),
		width:    sx,
		height:   sy,
	}
}

// Width of the grid.
func (g *EntityGrid) Width() int { return g.width }

// Height of the grid.
func (g *EntityGrid) Height() int { return g.height }

// Get the tree entity at the given coordinates.
func (g *EntityGrid) Get(x, y int) ecs.Entity {
	return g.entities[y+g.height*x]
}

// Set the tree entity at the given coordinates.
func (g *EntityGrid) Set(x, y int, e ecs.Entity) {
	g.entities[y+g.height*x] = e
}
