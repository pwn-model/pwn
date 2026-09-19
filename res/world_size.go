package res

// WorldSize resource.
type WorldSize struct {
	width        int // Width of the world in tree diameters.
	height       int // Height of the world in tree diameters.
	cellSize     int // Cell size of the tree grid in meters.
	gridCellSize int // Cell size of the spatial grid in meters.
	resolution   int // Cell size of the spatial grid in tree diameters.
}

// NewWorldSize creates a new [WorldSize].
//
// Arguments are:
// - World width in meters
// - World height in meters
// - Cell size of the tree grid in meters
// - Cell size of the spatial grid in meters
func NewWorldSize(width, height, cellSize, gridCellSize int) WorldSize {
	if width%cellSize != 0 {
		panic("World width must be a multiple of cellSize")
	}
	if height%cellSize != 0 {
		panic("World height must be a multiple of cellSize")
	}
	if gridCellSize%cellSize != 0 {
		panic("Space grid cell size must be a multiple of cellSize")
	}
	return WorldSize{
		width:        width / cellSize,
		height:       height / cellSize,
		cellSize:     cellSize,
		gridCellSize: gridCellSize,
		resolution:   gridCellSize / cellSize,
	}
}

// Width of the world in tree diameters.
func (s *WorldSize) Width() int {
	return s.width
}

// Height of the world in tree diameters.s
func (s *WorldSize) Height() int {
	return s.height
}

// CellSize of the tree grid in meters.
func (s *WorldSize) CellSize() int {
	return s.cellSize
}

// GridCellSize of the spatial grid in meters.
func (s *WorldSize) GridCellSize() int {
	return s.gridCellSize
}

// Resolution is the cell size of the spatial grid in tree diameters.
func (s *WorldSize) Resolution() int {
	return s.resolution
}
