package sys

import (
	"math"
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func setupTreeAttractionWorld(t *testing.T) (*ecs.World, *ecs.Map1[comp.Position]) {
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

func TestTreeAttractionSingleSourceDecaysMultiplicatively(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	halfDistance := 50 * math.Ln2
	s := TreeAttraction{TickOfYear: 0, HalfDistance: halfDistance, DensityRadius: 20, DensityWeight: 0}
	s.Initialize(world)
	s.Update(world)

	decay := math.Exp(-10.0 * math.Ln2 / halfDistance)
	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	// A single isolated tree seeds at count^0=1; 10 cells away (a purely
	// orthogonal offset, so the chamfer sweep is exact here, not
	// approximate), the value should be exactly decay^10 -- confirming
	// fillGrid decays multiplicatively rather than subtracting a fixed
	// cost per step.
	assert.InDelta(t, math.Pow(decay, 10), grid.Get(60, 25), 1e-9)
}

func TestTreeAttractionHasNoHardCutoff(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	s := TreeAttraction{TickOfYear: 0, HalfDistance: 10 * math.Ln2, DensityRadius: 20, DensityWeight: 0}
	s.Initialize(world)
	s.Update(world)

	// Multiplicative decay never hits an exact 0, however far from any
	// source: even at the far opposite corner of the world, the value must
	// be strictly positive.
	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	assert.Greater(t, grid.Get(199, 49), 0.0)
}

func TestTreeAttractionZeroWeightFavoursCloserIsolatedTree(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	placeIsolatedTreeAndCluster(posMap)

	halfDistance := 200 * math.Ln2
	s := TreeAttraction{TickOfYear: 0, HalfDistance: halfDistance, DensityRadius: 20, DensityWeight: 0}
	s.Initialize(world)
	s.Update(world)

	// At weight 0 every source seeds at 1 regardless of clustering, so
	// max-relaxation just tracks the nearest source: the closer isolated
	// tree (10 cells away) must win over the farther cluster (39 cells to
	// its nearest tree).
	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	decay := math.Exp(-10.0 * math.Ln2 / halfDistance)
	assert.InDelta(t, math.Pow(decay, 10), grid.Get(60, 25), 1e-9)
}

func TestTreeAttractionWeightLetsFartherDenserClusterWin(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	placeIsolatedTreeAndCluster(posMap)

	halfDistance := 200 * math.Ln2
	s := TreeAttraction{TickOfYear: 0, HalfDistance: halfDistance, DensityRadius: 20, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	// At weight 1, each seed is its occupancy fraction (local count / cells
	// in a DensityRadius window): the cluster's 9/25 versus the isolated
	// tree's 1/25 (DensityRadius=20, CellSize=10 -> a 5x5=25-cell window).
	// The cluster's contribution ((9/25)*decay^39) can still out-reach the
	// isolated tree's ((1/25)*decay^10) despite being 3x farther away, and
	// despite both being scaled down by the same normalization -- a
	// deterministic "pick the highest neighbour" consumer is now pulled
	// towards the farther, denser cluster instead of the closer lone tree.
	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	decay := math.Exp(-10.0 * math.Ln2 / halfDistance)
	maxCount := (2.0*2.0 + 1.0) * (2.0*2.0 + 1.0) // DensityRadius=20, CellSize=10 -> densityRadiusCells=2.
	isolatedValue := (1.0 / maxCount) * math.Pow(decay, 10)
	clusterValue := (9.0 / maxCount) * math.Pow(decay, 39)
	assert.Greater(t, clusterValue, isolatedValue, "sanity check: chosen HalfDistance must make the cluster the stronger source")
	assert.InDelta(t, clusterValue, grid.Get(60, 25), 1e-9)
}

// A fully-occupied patch (a tree in every cell of its own DensityRadius
// window) has an occupancy fraction of exactly 1, so its seed is exactly
// 1 regardless of DensityWeight -- and since fillGrid's max-relaxation can
// never exceed the largest seed anywhere in the grid, no cell of the
// resulting field can exceed 1 either, however densely populated the world
// is or how high DensityWeight is set.
func TestTreeAttractionFieldNeverExceedsOne(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			posMap.NewEntity(&comp.Position{X: 50 + dx, Y: 25 + dy})
		}
	}

	s := TreeAttraction{TickOfYear: 0, HalfDistance: 200 * math.Ln2, DensityRadius: 20, DensityWeight: 5}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	assert.InDelta(t, 1.0, grid.Get(50, 25), 1e-9, "a fully-occupied cell's seed must be exactly 1")

	max := 0.0
	for x := range grid.Width() {
		for y := range grid.Height() {
			if v := grid.Get(x, y); v > max {
				max = v
			}
		}
	}
	assert.LessOrEqual(t, max, 1.0+1e-9, "no cell may exceed a seed value of 1")
}

// At DensityWeight 0, fillFromQuery takes a fast path that seeds every
// source at 1 directly, without ever consulting DensityRadius -- so an
// invalid DensityRadius (here, not a multiple of the world's 10m cell size)
// must not panic during Initialize.
func TestTreeAttractionZeroWeightSkipsDensityRadiusValidation(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	s := TreeAttraction{TickOfYear: 0, HalfDistance: 50 * math.Ln2, DensityRadius: 7, DensityWeight: 0}
	assert.NotPanics(t, func() { s.Initialize(world) })
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	assert.InDelta(t, 1.0, grid.Get(50, 25), 1e-9)
}

func TestTreeAttractionSkipsWrongTickOfYear(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	s := TreeAttraction{TickOfYear: 5, HalfDistance: 200 * math.Ln2, DensityRadius: 20, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world)
	assert.Equal(t, 0.0, grid.Get(50, 25))
}
