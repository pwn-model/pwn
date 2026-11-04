package maps

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func setupSourceMapObserverWorld(t *testing.T, width, height int, damagedTrees bool) (*ecs.World, *TreeAttractionSourceMap, *res.Grid[uint32]) {
	t.Helper()

	a := app.New()
	source := res.NewGrid[uint32](width, height, 10)
	source.Fill(res.NoCell)
	attraction := res.NewGrid[float64](width, height, 10)
	presence := res.NewGrid[bool](width, height, 10)

	var grid *res.Grid[uint32]
	if damagedTrees {
		r := res.DamagedTreeAttraction{Attraction: attraction, Presence: presence, Source: source}
		ecs.AddResource(a.World, &r)
		grid = &r.Source
	} else {
		r := res.HealthyTreeAttraction{Attraction: attraction, Presence: presence, Source: source}
		ecs.AddResource(a.World, &r)
		grid = &r.Source
	}

	o := TreeAttractionSourceMap{DamagedTrees: damagedTrees}
	o.Initialize(a.World)

	return a.World, &o, grid
}

func TestTreeAttractionSourceMapDimsMatchSourceGrid(t *testing.T) {
	_, o, _ := setupSourceMapObserverWorld(t, 5, 3, false)

	width, height := o.Dims()
	assert.Equal(t, 5, width)
	assert.Equal(t, 3, height)
}

func TestTreeAttractionSourceMapNoSource(t *testing.T) {
	world, o, _ := setupSourceMapObserverWorld(t, 4, 4, false)

	for _, v := range o.Values(world) {
		assert.Equal(t, -1.0, v)
	}
}

func TestTreeAttractionSourceMapStableValues(t *testing.T) {
	for _, damagedTrees := range []bool{false, true} {
		world, o, source := setupSourceMapObserverWorld(t, 4, 4, damagedTrees)
		w, _ := o.Dims()
		at := func(values []float64, x, y int) float64 { return values[y*w+x] }

		a := source.Index(0, 0)
		b := source.Index(1, 0)
		source.Set(0, 0, a)
		source.Set(0, 1, a)
		source.Set(1, 0, b)
		source.Set(1, 1, b)

		values := o.Values(world)

		assert.Equal(t, at(values, 0, 0), at(values, 0, 1))
		assert.Equal(t, at(values, 1, 0), at(values, 1, 1))
		assert.NotEqual(t, at(values, 0, 0), at(values, 1, 0))
		assert.True(t, at(values, 0, 0) >= 0 && at(values, 0, 0) < 1)
		assert.True(t, at(values, 1, 0) >= 0 && at(values, 1, 0) < 1)
		assert.Equal(t, -1.0, at(values, 2, 2))

		// Updated in place on each call.
		first := at(values, 0, 0)
		source.Set(2, 2, a)
		assert.Equal(t, first, at(o.Values(world), 2, 2))
	}
}

func TestTreeAttractionSourceMapScattersNeighbouringIndices(t *testing.T) {
	// Consecutive indices must not map to similar values, or neighbouring
	// patches would get near-identical colors.
	seen := map[float64]bool{}
	sumDiff := 0.0
	prev := hashToUnit(0)
	for i := range uint32(1000) {
		v := hashToUnit(i)
		assert.True(t, v >= 0 && v < 1)
		seen[v] = true
		if i > 0 {
			d := v - prev
			if d < 0 {
				d = -d
			}
			sumDiff += d
		}
		prev = v
	}
	assert.Len(t, seen, 1000)
	assert.Greater(t, sumDiff/999, 0.25)
}
