package res

import "math"

// NoCell is a placeholder cell index for the absence of a cell, e.g. of an
// attraction source. See [Grid.Index].
const NoCell uint32 = math.MaxUint32

// Grid resource.
//
// Values are stored column-major (consecutive y for fixed x are contiguous),
// so that column-wise iteration is cache-friendly for large grids.
type Grid[T any] struct {
	values   []T
	width    int
	height   int
	cellSize int
}

// NewGrid creates a new [Grid] of the given size.
func NewGrid[T any](sx, sy, cellSize int) Grid[T] {
	return Grid[T]{
		values:   make([]T, sx*sy),
		width:    sx,
		height:   sy,
		cellSize: cellSize,
	}
}

// Width of the grid in cells.
func (g *Grid[T]) Width() int { return g.width }

// Height of the grid in cells.
func (g *Grid[T]) Height() int { return g.height }

// CellSize of the grid in meters.
func (g *Grid[T]) CellSize() int { return g.cellSize }

// Get the tree entity at the given coordinates.
func (g *Grid[T]) Get(x, y int) T {
	return g.values[y+g.height*x]
}

// Set the tree entity at the given coordinates.
func (g *Grid[T]) Set(x, y int, e T) {
	g.values[y+g.height*x] = e
}

// Index returns the linear index of the cell at (x, y), as used for the
// grid's own storage. Only valid for grids with less than [NoCell] cells,
// so that NoCell is never a valid index.
func (g *Grid[T]) Index(x, y int) uint32 {
	return uint32(y + g.height*x)
}

// Coords returns the coordinates of the cell with the given linear index.
// Inverse of [Grid.Index].
func (g *Grid[T]) Coords(idx uint32) (int, int) {
	i := int(idx)
	return i / g.height, i % g.height
}

// Fill the grid with the given value.
func (g *Grid[T]) Fill(v T) {
	for i := range g.values {
		g.values[i] = v
	}
}
