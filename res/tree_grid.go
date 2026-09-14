package res

import "github.com/mlange-42/ark/ecs"

// TreeGrid resource.
//
// Trees are stored column-major (consecutive y for fixed x are contiguous),
// so that column-wise iteration is cache-friendly for large grids.
type TreeGrid struct {
	trees  []ecs.Entity
	width  int
	height int
}

// NewTreeGrid creates a new [TreeGrid] of the given size.
func NewTreeGrid(sx, sy int) TreeGrid {
	return TreeGrid{
		trees:  make([]ecs.Entity, sx*sy),
		width:  sx,
		height: sy,
	}
}

// Width of the grid.
func (g *TreeGrid) Width() int { return g.width }

// Height of the grid.
func (g *TreeGrid) Height() int { return g.height }

// Get the tree entity at the given coordinates.
func (g *TreeGrid) Get(x, y int) ecs.Entity {
	return g.trees[y+g.height*x]
}

// Set the tree entity at the given coordinates.
func (g *TreeGrid) Set(x, y int, e ecs.Entity) {
	g.trees[y+g.height*x] = e
}
