package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func setupTreeAttractionDensityPeakWorld(t *testing.T) (*ecs.World, *ecs.Map1[comp.Position]) {
	t.Helper()

	a := app.New()
	ws := res.NewWorldSize(2000, 500, 10, 100)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 0})

	return a.World, ecs.NewMap1[comp.Position](a.World)
}

// A closer isolated tree at (50, 25) and a farther 3x3 cluster of 9 trees
// around (100, 25), with a query point at (60, 25): 10 cells from the
// isolated tree, 39 cells from the cluster's nearest edge.
func placeIsolatedTreeAndCluster(posMap *ecs.Map1[comp.Position]) {
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			posMap.NewEntity(&comp.Position{X: 100 + dx, Y: 25 + dy})
		}
	}
}

func TestTreeAttractionDensityPeakZeroWeightMatchesPlainTreeAttraction(t *testing.T) {
	world, posMap := setupTreeAttractionDensityPeakWorld(t)
	placeIsolatedTreeAndCluster(posMap)

	s := TreeAttractionDensityPeak{TickOfYear: 0, Radius: 200, DensityRadius: 20, DensityWeight: 0}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	// At weight 0 every source peaks at Radius/CellSize=20 regardless of
	// clustering, so the closer isolated tree (10 cells away) must win
	// over the farther cluster (39 cells to its nearest tree): the query
	// point's value must equal the plain "distance to nearest source"
	// result, 20-10=10.
	assert.InDelta(t, 10.0, grid.Get(60, 25), 1e-9)
}

func TestTreeAttractionDensityPeakLetsADenserFartherClusterWin(t *testing.T) {
	world, posMap := setupTreeAttractionDensityPeakWorld(t)
	placeIsolatedTreeAndCluster(posMap)

	s := TreeAttractionDensityPeak{TickOfYear: 0, Radius: 200, DensityRadius: 20, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	// At weight 1, the cluster's 9 local trees scale its peak to
	// 20*9=180, reaching the query point (39 cells away) with 180-39=141
	// -- far more than the isolated tree's fixed 20-10=10. A deterministic
	// "pick the highest neighbour" consumer would now be pulled towards
	// the farther, denser cluster instead of the closer lone tree, which
	// TreeAttraction/TreeAttractionExponential-style final rescaling could
	// never achieve (see the type's doc comment).
	assert.InDelta(t, 141.0, grid.Get(60, 25), 1e-9)
}

func TestTreeAttractionDensityPeakSkipsWrongTickOfYear(t *testing.T) {
	world, posMap := setupTreeAttractionDensityPeakWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	s := TreeAttractionDensityPeak{TickOfYear: 5, Radius: 200, DensityRadius: 20, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	assert.Equal(t, 0.0, grid.Get(50, 25))
}
