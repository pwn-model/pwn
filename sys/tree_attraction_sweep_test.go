package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func setupTreeAttractionSweepWorld(t *testing.T) (*ecs.World, *ecs.Map1[comp.Position]) {
	t.Helper()

	a := app.New()
	ws := res.NewWorldSize(1000, 100, 10, 100)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 0})

	return a.World, ecs.NewMap1[comp.Position](a.World)
}

func TestTreeAttractionSweepClusterMoreAttractiveThanSingleTree(t *testing.T) {
	world, posMap := setupTreeAttractionSweepWorld(t)

	// A small cluster of healthy trees around (10, 5).
	for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
		posMap.NewEntity(&comp.Position{X: 10 + d[0], Y: 5 + d[1]})
	}
	// A single healthy tree, far enough away not to overlap the cluster's blur.
	posMap.NewEntity(&comp.Position{X: 80, Y: 5})

	s := TreeAttractionSweep{TickOfYear: 0, Scale: 10, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	atCluster := grid.Get(10, 5)
	atSingle := grid.Get(80, 5)

	assert.Greater(t, atCluster, atSingle, "a cluster of trees must be more attractive than a single tree")
}

func TestTreeAttractionSweepDensityWeightZeroIgnoresDensity(t *testing.T) {
	world, posMap := setupTreeAttractionSweepWorld(t)

	for _, d := range [][2]int{{0, 0}, {1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
		posMap.NewEntity(&comp.Position{X: 10 + d[0], Y: 5 + d[1]})
	}
	posMap.NewEntity(&comp.Position{X: 80, Y: 5})

	s := TreeAttractionSweep{TickOfYear: 0, Scale: 10, DensityWeight: 0}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	assert.InDelta(t, grid.Get(10, 5), grid.Get(80, 5), 1e-9,
		"with DensityWeight 0, a cluster and a single tree must look equally attractive")
	assert.Equal(t, 1.0, grid.Get(10, 5))
}

func TestTreeAttractionSweepSkipsWrongTickOfYear(t *testing.T) {
	world, posMap := setupTreeAttractionSweepWorld(t)
	posMap.NewEntity(&comp.Position{X: 10, Y: 5})

	s := TreeAttractionSweep{TickOfYear: 5, Scale: 10, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	assert.Equal(t, 0.0, grid.Get(10, 5))
}

func TestTreeAttractionSweepIsApproximatelyRadiallySymmetric(t *testing.T) {
	a := app.New()
	ws := res.NewWorldSize(400, 400, 10, 100)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 0})
	posMap := ecs.NewMap1[comp.Position](a.World)

	posMap.NewEntity(&comp.Position{X: 20, Y: 20})

	s := TreeAttractionSweep{TickOfYear: 0, Scale: 30, DensityWeight: 1}
	s.Initialize(a.World)
	s.Update(a.World)

	grid := ecs.GetResource[res.HealthyTreeAttraction](a.World)

	// A plain separable exponential decay would be diamond-shaped: its
	// value depends on |dx|+|dy| (Manhattan distance), so an axis-aligned
	// offset (Manhattan == Euclidean) would score much higher than a
	// diagonal offset at a similar or even slightly smaller Euclidean
	// distance but larger Manhattan distance. An isotropic field should
	// instead score the diagonal offset at least as high.
	axis := grid.Get(29, 20)     // offset (9, 0): Euclidean 9.00, Manhattan 9.
	diagonal := grid.Get(26, 26) // offset (6, 6): Euclidean 8.49, Manhattan 12.

	assert.GreaterOrEqual(t, diagonal, axis*0.9,
		"a diagonal offset at a similar Euclidean distance must not score much lower than an axis-aligned one")
}
