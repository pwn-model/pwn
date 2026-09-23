package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func setupTreeAttractionDensityRecursiveWorld(t *testing.T) (*ecs.World, *ecs.Map1[comp.Position]) {
	t.Helper()

	a := app.New()
	ws := res.NewWorldSize(2000, 500, 10, 100)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 0})

	return a.World, ecs.NewMap1[comp.Position](a.World)
}

// runTreeAttractionDensityRecursiveContribution runs the system with only
// the given trees present, and returns the resulting healthy-attraction
// value at the query point (60, 25) -- since propagation is a linear sum of
// independent per-source contributions, this measures exactly one source
// configuration's own contribution there, in isolation from any other
// source.
func runTreeAttractionDensityRecursiveContribution(t *testing.T, weight float64, positions [][2]int) float64 {
	t.Helper()

	world, posMap := setupTreeAttractionDensityRecursiveWorld(t)
	for _, p := range positions {
		posMap.NewEntity(&comp.Position{X: p[0], Y: p[1]})
	}

	s := TreeAttractionDensityRecursive{TickOfYear: 0, Scale: 50, DensityRadius: 20, DensityWeight: weight}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	return grid.Get(60, 25)
}

var isolatedTree = [][2]int{{50, 25}}

var cluster3x3 = func() [][2]int {
	var ps [][2]int
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			ps = append(ps, [2]int{90 + dx, 25 + dy})
		}
	}
	return ps
}()

// At the query point (60, 25) -- 10 cells from the isolated tree, 30 cells
// from the cluster's nearest edge -- proximity alone favours the isolated
// tree. With DensityWeight 0, every source seeds at 1 regardless of local
// count, so that's exactly what happens.
func TestTreeAttractionDensityRecursiveZeroWeightFavoursCloserIsolatedTree(t *testing.T) {
	isoContribution := runTreeAttractionDensityRecursiveContribution(t, 0, isolatedTree)
	clusterContribution := runTreeAttractionDensityRecursiveContribution(t, 0, cluster3x3)

	assert.Greater(t, isoContribution, clusterContribution,
		"at DensityWeight 0, the closer isolated tree must contribute more than the farther cluster")
}

// Raising DensityWeight lets the cluster's 9 local trees (each seeded at
// 9^weight instead of 1) out-contribute the closer isolated tree, despite
// being 3x farther away -- exactly the deterministic-comparison-relevant
// effect a final DensityWeight exponent (TreeAttractionRecursive,
// TreeAttractionSweep) can never produce.
func TestTreeAttractionDensityRecursiveWeightLetsFartherDenserClusterWin(t *testing.T) {
	isoContribution := runTreeAttractionDensityRecursiveContribution(t, 1, isolatedTree)
	clusterContribution := runTreeAttractionDensityRecursiveContribution(t, 1, cluster3x3)

	assert.Greater(t, clusterContribution, isoContribution,
		"at DensityWeight 1, the farther but denser cluster must out-contribute the closer isolated tree")
}

func TestTreeAttractionDensityRecursiveSkipsWrongTickOfYear(t *testing.T) {
	world, posMap := setupTreeAttractionDensityRecursiveWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	s := TreeAttractionDensityRecursive{TickOfYear: 5, Scale: 50, DensityRadius: 20, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	assert.Equal(t, 0.0, grid.Get(50, 25))
}
