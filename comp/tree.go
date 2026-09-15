package comp

import "github.com/mlange-42/ark/ecs"

// Position component with integer cell coordinates.
type Position struct {
	X int
	Y int
}

// InCell component for relations of trees to their containing spatial grid cells.
type InCell struct {
	ecs.RelationMarker
}
