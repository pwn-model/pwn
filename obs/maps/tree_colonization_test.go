package maps

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func setupMapObserverWorld(t *testing.T, width, height int) *ecs.World {
	t.Helper()

	a := app.New()
	ws := res.NewWorldSize(width, height, 10, 10)
	ecs.AddResource(a.World, &ws)

	return a.World
}

func TestTreeColonizationRejectsNonMultipleCellSize(t *testing.T) {
	a := app.New()
	ws := res.NewWorldSize(100, 100, 10, 10)
	ecs.AddResource(a.World, &ws)

	o := TreeColonization{CellSize: 25}
	assert.Panics(t, func() { o.Initialize(a.World) })
}

func TestTreeColonizationDimsAreWorldSizeDividedByCellSize(t *testing.T) {
	world := setupMapObserverWorld(t, 100, 40)

	o := TreeColonization{CellSize: 30}
	o.Initialize(world)

	width, height := o.Dims()
	assert.Equal(t, 4, width)
	assert.Equal(t, 2, height)
}

func TestTreeColonizationCountsColonizedTreesPerCell(t *testing.T) {
	world := setupMapObserverWorld(t, 100, 100)

	o := TreeColonization{CellSize: 30}
	o.Initialize(world)

	colonizedMap := ecs.NewMap2[comp.Position, comp.Colonized](world)
	// Both land in map cell (0, 0) (tree-grid coords 0..2).
	colonizedMap.NewEntity(&comp.Position{X: 0, Y: 0}, &comp.Colonized{})
	colonizedMap.NewEntity(&comp.Position{X: 1, Y: 1}, &comp.Colonized{})
	// Lands in map cell (1, 1) (tree-grid coords 3..5).
	colonizedMap.NewEntity(&comp.Position{X: 3, Y: 3}, &comp.Colonized{})

	// Damaged but not colonized: must not be counted.
	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	damagedMap.NewEntity(&comp.Position{X: 8, Y: 8}, &comp.Damaged{})

	values := o.Values(world)
	width, _ := o.Dims()

	assert.Equal(t, 2.0, values[0*width+0])
	assert.Equal(t, 1.0, values[1*width+1])

	total := 0.0
	for _, v := range values {
		total += v
	}
	assert.Equal(t, 3.0, total)
}

func TestTreeColonizationDoesNotCarryCountsOverBetweenCalls(t *testing.T) {
	world := setupMapObserverWorld(t, 100, 100)

	o := TreeColonization{CellSize: 30}
	o.Initialize(world)

	colonizedMap := ecs.NewMap2[comp.Position, comp.Colonized](world)
	entity := colonizedMap.NewEntity(&comp.Position{X: 0, Y: 0}, &comp.Colonized{})

	width, _ := o.Dims()

	first := o.Values(world)
	assert.Equal(t, 1.0, first[0*width+0])

	colonizedRemoveMap := ecs.NewMap1[comp.Colonized](world)
	colonizedRemoveMap.Remove(entity)

	second := o.Values(world)
	assert.Equal(t, 0.0, second[0*width+0])
}
