package res

import "github.com/mlange-42/ark/ecs"

// TreeGrid resource for the small-scale trees grid.
type TreeGrid struct {
	Grid[ecs.Entity]
}

// HealthyTreeAttraction resource.
type HealthyTreeAttraction struct {
	Grid[float64]
}

// DamagedTreeAttraction resource.
type DamagedTreeAttraction struct {
	Grid[float64]
}

// HealthyTreeAttractionCoarse resource, on the coarse grid of the
// CoarseTreeAttraction system.
type HealthyTreeAttractionCoarse struct {
	Grid[float64]
}

// DamagedTreeAttractionCoarse resource, on the coarse grid of the
// CoarseTreeAttraction system.
type DamagedTreeAttractionCoarse struct {
	Grid[float64]
}

// FeedingInfectedBeetles resource.
type FeedingInfectedBeetles struct {
	Grid[uint32]
}
