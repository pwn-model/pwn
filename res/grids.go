package res

import "github.com/mlange-42/ark/ecs"

// TreeGrid resource for the small-scale trees grid.
type TreeGrid struct {
	Grid[ecs.Entity]
}

// HealthyTreeAttraction resource.
type HealthyTreeAttraction struct {
	// Attraction field of healthy trees, updated once per year.
	Attraction Grid[float64]
	// Presence of healthy trees per cell, updated together with Attraction.
	Presence Grid[bool]
}

// DamagedTreeAttraction resource.
type DamagedTreeAttraction struct {
	// Attraction field of damaged trees, updated once per year.
	Attraction Grid[float64]
	// Presence of damaged trees per cell, updated together with Attraction.
	Presence Grid[bool]
}

// FeedingInfectedBeetles resource.
type FeedingInfectedBeetles struct {
	Grid[uint32]
}
