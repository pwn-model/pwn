package res

import "github.com/pwn-model/pwn/config"

func init() {
	config.RegisterResource(func(c WorldSizeConfig) WorldSize {
		return NewWorldSize(c.Width, c.Height, c.CellSize)
	})
}

// WorldSizeConfig holds the arguments of [NewWorldSize], for use as a
// config file's "resources" entry.
type WorldSizeConfig struct {
	Width    int `yaml:"width"`
	Height   int `yaml:"height"`
	CellSize int `yaml:"cell_size"`
}

// WorldSize resource.
type WorldSize struct {
	width    int // Width of the world in tree diameters.
	height   int // Height of the world in tree diameters.
	cellSize int // Cell size of the tree grid in meters.
}

// NewWorldSize creates a new [WorldSize].
//
// Arguments are:
// - World width in meters
// - World height in meters
// - Cell size of the tree grid in meters
func NewWorldSize(width, height, cellSize int) WorldSize {
	if width%cellSize != 0 {
		panic("World width must be a multiple of cellSize")
	}
	if height%cellSize != 0 {
		panic("World height must be a multiple of cellSize")
	}
	return WorldSize{
		width:    width / cellSize,
		height:   height / cellSize,
		cellSize: cellSize,
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
