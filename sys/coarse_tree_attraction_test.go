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

// setupCoarseTreeAttractionWorld creates a world of the given size in
// meters, with a 10m base cell size.
func setupCoarseTreeAttractionWorld(t *testing.T, width, height int) (*ecs.World, *ecs.Map1[comp.Position]) {
	t.Helper()

	a := app.New()
	ws := res.NewWorldSize(width, height, 10)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 0})

	return a.World, ecs.NewMap1[comp.Position](a.World)
}

// placeTreeBlock places a tree in every base cell of the given rectangle.
func placeTreeBlock(posMap *ecs.Map1[comp.Position], x0, y0, w, h int) {
	for x := x0; x < x0+w; x++ {
		for y := y0; y < y0+h; y++ {
			posMap.NewEntity(&comp.Position{X: x, Y: y})
		}
	}
}

func TestCoarseTreeAttractionGridDimensions(t *testing.T) {
	// 2050m x 500m at 100m: 20.5 x 5 coarse cells, rounded up to 21 x 5.
	world, _ := setupCoarseTreeAttractionWorld(t, 2050, 500)

	s := CoarseTreeAttraction{TickOfYear: 0, CellSize: 100, HalfDistance: 100, DensityWeight: 1}
	s.Initialize(world)

	grid := ecs.GetResource[res.HealthyTreeAttractionCoarse](world)
	assert.Equal(t, 21, grid.Width())
	assert.Equal(t, 5, grid.Height())
	assert.Equal(t, 100, grid.CellSize())
}

// A coarse cell's seed is its tree count divided by the number of base
// cells it covers, raised to DensityWeight.
func TestCoarseTreeAttractionSeedIsOccupancyFraction(t *testing.T) {
	for _, weight := range []float64{0, 1, 2} {
		world, posMap := setupCoarseTreeAttractionWorld(t, 2000, 500)
		// 25 trees in coarse cell (0, 0), which covers 10 x 10 base cells.
		placeTreeBlock(posMap, 0, 0, 5, 5)

		s := CoarseTreeAttraction{TickOfYear: 0, CellSize: 100, HalfDistance: 100, DensityWeight: weight}
		s.Initialize(world)
		s.Update(world)

		grid := ecs.GetResource[res.HealthyTreeAttractionCoarse](world)
		assert.InDelta(t, math.Pow(0.25, weight), grid.Get(0, 0), 1e-12, "DensityWeight %v", weight)
	}
}

// A partial coarse cell along the world's edge is normalized like any other
// cell, by the number of base cells in a full coarse cell, so even a fully
// occupied one stays below 1.
func TestCoarseTreeAttractionPartialEdgeCell(t *testing.T) {
	// 205 x 50 base cells; the last coarse column (20) covers only base
	// columns 200..204, i.e. 5 x 10 of a coarse cell's 10 x 10 base cells.
	world, posMap := setupCoarseTreeAttractionWorld(t, 2050, 500)
	placeTreeBlock(posMap, 200, 0, 5, 10)

	s := CoarseTreeAttraction{TickOfYear: 0, CellSize: 100, HalfDistance: 100, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttractionCoarse](world)
	assert.InDelta(t, 0.5, grid.Get(20, 0), 1e-12)
}

// HalfDistance is in meters, independent of the coarse cell size.
func TestCoarseTreeAttractionHalfDistanceInMeters(t *testing.T) {
	world, posMap := setupCoarseTreeAttractionWorld(t, 2000, 500)
	posMap.NewEntity(&comp.Position{X: 55, Y: 25}) // coarse cell (5, 2)

	s := CoarseTreeAttraction{TickOfYear: 0, CellSize: 100, HalfDistance: 300, DensityWeight: 0}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttractionCoarse](world)
	assert.InDelta(t, 1.0, grid.Get(5, 2), 1e-12)
	// 3 coarse cells = 300m away: half the seed.
	assert.InDelta(t, 0.5, grid.Get(8, 2), 1e-12)
	assert.InDelta(t, 0.5, grid.Get(2, 2), 1e-12)
}

func TestCoarseTreeAttractionHealthyAndDamagedInstancesAreIndependent(t *testing.T) {
	world, posMap := setupCoarseTreeAttractionWorld(t, 2000, 500)
	posMap.NewEntity(&comp.Position{X: 55, Y: 25}) // coarse cell (5, 2)

	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	damagedMap.NewEntity(&comp.Position{X: 125, Y: 25}, &comp.Damaged{}) // coarse cell (12, 2)

	healthy := CoarseTreeAttraction{TickOfYear: 0, CellSize: 100, HalfDistance: 100, DensityWeight: 0}
	healthy.Initialize(world)
	damaged := CoarseTreeAttraction{TickOfYear: 0, DamagedTrees: true, CellSize: 100, HalfDistance: 100, DensityWeight: 0}
	damaged.Initialize(world)

	healthy.Update(world)
	damaged.Update(world)

	healthyGrid := ecs.GetResource[res.HealthyTreeAttractionCoarse](world)
	damagedGrid := ecs.GetResource[res.DamagedTreeAttractionCoarse](world)

	assert.InDelta(t, 1.0, healthyGrid.Get(5, 2), 1e-12)
	assert.InDelta(t, math.Pow(0.5, 7), healthyGrid.Get(12, 2), 1e-12)
	assert.InDelta(t, 1.0, damagedGrid.Get(12, 2), 1e-12)
	assert.InDelta(t, math.Pow(0.5, 7), damagedGrid.Get(5, 2), 1e-12)
}

func TestCoarseTreeAttractionPanicsOnInvalidCellSize(t *testing.T) {
	world, _ := setupCoarseTreeAttractionWorld(t, 2000, 500)

	s := CoarseTreeAttraction{TickOfYear: 0, CellSize: 105, HalfDistance: 100, DensityWeight: 1}
	assert.Panics(t, func() { s.Initialize(world) })
}

func TestCoarseTreeAttractionSkipsWrongTickOfYear(t *testing.T) {
	world, posMap := setupCoarseTreeAttractionWorld(t, 2000, 500)
	posMap.NewEntity(&comp.Position{X: 55, Y: 25})

	s := CoarseTreeAttraction{TickOfYear: 5, CellSize: 100, HalfDistance: 100, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttractionCoarse](world)
	assert.Equal(t, 0.0, grid.Get(5, 2))
}
