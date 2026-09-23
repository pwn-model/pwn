package res

import "github.com/mlange-42/ark/ecs"

// TreeGrid resource for the small-scale trees grid.
type TreeGrid struct {
	Grid[ecs.Entity]
}

// SpaceGrid resource for the large-scale spatial grid.
type SpaceGrid struct {
	Grid[ecs.Entity]
}

// HealthyTreeAttractionNear resource.
type HealthyTreeAttractionNear struct {
	Grid[float64]
}

// HealthyTreeAttractionFar resource.
type HealthyTreeAttractionFar struct {
	Grid[float64]
}

// DamagedTreeAttractionNear resource.
type DamagedTreeAttractionNear struct {
	Grid[float64]
}

// DamagedTreeAttractionFar resource.
type DamagedTreeAttractionFar struct {
	Grid[float64]
}
