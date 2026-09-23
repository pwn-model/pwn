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

// HealthyTreeAttraction resource.
type HealthyTreeAttraction struct {
	Grid[float64]
}

// DamagedTreeAttraction resource.
type DamagedTreeAttraction struct {
	Grid[float64]
}
