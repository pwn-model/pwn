package sys

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

// initTreeStandsWorld runs InitGrids and InitTreeStands on a 1000 x 500 m
// world with 10 m cells.
func initTreeStandsWorld(stands []TreeStand) *ecs.World {
	app := app.New()

	ws := res.NewWorldSize(1000, 500, 10)
	ecs.AddResource(app.World, &ws)

	gs := InitGrids{}
	gs.Initialize(app.World)

	s := InitTreeStands{Stands: stands}
	s.Initialize(app.World)

	return app.World
}

// fullStand is a stand with a tree in every cell, none damaged.
func fullStand(xMin, yMin, xMax, yMax float64) TreeStand {
	return TreeStand{XMin: xMin, YMin: yMin, XMax: xMax, YMax: yMax, TreeProbability: 1.0}
}

// treePositions returns the positions of all trees with the given components.
func treePositions(world *ecs.World, with ...ecs.Comp) []comp.Position {
	var positions []comp.Position
	q := ecs.NewFilter1[comp.Position](world).With(with...).Query()
	for q.Next() {
		positions = append(positions, *q.Get())
	}
	return positions
}

func TestInitTreeStandsBounds(t *testing.T) {
	// Cells with centers in [100, 300) x [50, 250): x 10..29, y 5..24.
	world := initTreeStandsWorld([]TreeStand{fullStand(100, 50, 300, 250)})
	grid := ecs.GetResource[res.TreeGrid](world)

	q := ecs.NewFilter1[comp.Position](world).Query()
	assert.Equal(t, 20*20, q.Count())
	for q.Next() {
		pos := q.Get()
		assert.True(t, pos.X >= 10 && pos.X <= 29, "x = %d", pos.X)
		assert.True(t, pos.Y >= 5 && pos.Y <= 24, "y = %d", pos.Y)
		assert.Equal(t, q.Entity(), grid.Get(pos.X, pos.Y))
	}
}

func TestInitTreeStandsOutsideWorld(t *testing.T) {
	// Clipped to the world: x 0..99, y 40..49.
	world := initTreeStandsWorld([]TreeStand{fullStand(-500, 400, 5000, 1e9)})
	assert.Len(t, treePositions(world), 100*10)

	world = initTreeStandsWorld([]TreeStand{fullStand(2000, 0, 3000, 500)})
	assert.Empty(t, treePositions(world))

	world = initTreeStandsWorld(nil)
	assert.Empty(t, treePositions(world))
}

func TestInitTreeStandsOverlap(t *testing.T) {
	// Second stand is empty and overwrites the right half of the first one.
	empty := fullStand(500, 0, 1000, 500)
	empty.TreeProbability = 0.0
	world := initTreeStandsWorld([]TreeStand{fullStand(0, 0, 1000, 500), empty})

	positions := treePositions(world)
	assert.Len(t, positions, 50*50)
	for _, pos := range positions {
		assert.Less(t, pos.X, 50)
	}

	// Second stand fully damaged, overwriting the center of the first one.
	damaged := fullStand(200, 100, 400, 300)
	damaged.DamagePrevalence = 1.0
	world = initTreeStandsWorld([]TreeStand{fullStand(0, 0, 1000, 500), damaged})

	assert.Len(t, treePositions(world), 100*50)
	positions = treePositions(world, ecs.C[comp.Damaged]())
	assert.Len(t, positions, 20*20)
	for _, pos := range positions {
		assert.True(t, pos.X >= 20 && pos.X <= 39, "x = %d", pos.X)
		assert.True(t, pos.Y >= 10 && pos.Y <= 29, "y = %d", pos.Y)
	}
}

func TestInitTreeStandsPrevalences(t *testing.T) {
	left := fullStand(0, 0, 500, 500)
	left.TreeProbability = 0.9
	right := TreeStand{
		XMin: 500, YMin: 0, XMax: 1000, YMax: 500,
		TreeProbability: 0.5, DamagePrevalence: 0.2, BeetlePrevalence: 0.3,
	}
	world := initTreeStandsWorld([]TreeStand{left, right})

	countHalves := func(positions []comp.Position) (l, r int) {
		for _, pos := range positions {
			if pos.X < 50 {
				l++
			} else {
				r++
			}
		}
		return
	}

	nLeft, nRight := countHalves(treePositions(world))
	leftDamaged, rightDamaged := countHalves(treePositions(world, ecs.C[comp.Damaged]()))
	_, rightColonized := countHalves(treePositions(world, ecs.C[comp.Colonized]()))
	_, rightColonizedDamaged := countHalves(treePositions(world, ecs.C[comp.Colonized](), ecs.C[comp.Damaged]()))

	cells := 50.0 * 50.0
	assert.InDelta(t, 0.9*cells, float64(nLeft), 0.05*cells)
	assert.InDelta(t, 0.5*cells, float64(nRight), 0.05*cells)
	assert.Equal(t, 0, leftDamaged)
	assert.InDelta(t, 0.2*float64(nRight), float64(rightDamaged), 0.25*0.2*float64(nRight))
	assert.InDelta(t, 0.06*float64(nRight), float64(rightColonized), 0.25*0.06*float64(nRight))
	// Colonized trees are also damaged.
	assert.Equal(t, rightColonized, rightColonizedDamaged)
}
