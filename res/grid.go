package res

// Grid resource.
//
// Values are stored column-major (consecutive y for fixed x are contiguous),
// so that column-wise iteration is cache-friendly for large grids.
type Grid[T any] struct {
	values []T
	width  int
	height int
}

// NewGrid creates a new [Grid] of the given size.
func NewGrid[T any](sx, sy int) Grid[T] {
	return Grid[T]{
		values: make([]T, sx*sy),
		width:  sx,
		height: sy,
	}
}

// Width of the grid.
func (g *Grid[T]) Width() int { return g.width }

// Height of the grid.
func (g *Grid[T]) Height() int { return g.height }

// Get the tree entity at the given coordinates.
func (g *Grid[T]) Get(x, y int) T {
	return g.values[y+g.height*x]
}

// Set the tree entity at the given coordinates.
func (g *Grid[T]) Set(x, y int, e T) {
	g.values[y+g.height*x] = e
}
