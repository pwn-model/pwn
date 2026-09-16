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

// Damaged marks a tree as damaged in the perception of the vector.
type Damaged struct{}

// NematodeInfected marks a tree as infested with PWN.
type NematodeInfected struct{}
