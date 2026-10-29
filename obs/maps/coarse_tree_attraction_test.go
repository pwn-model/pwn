package maps

import (
	"math"
	"testing"

	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

func TestCoarseTreeAttractionMapReportsLogOfSelectedField(t *testing.T) {
	world := setupMapObserverWorld(t, 100, 100)

	healthy := res.NewGrid[float64](3, 2, 50)
	damaged := res.NewGrid[float64](3, 2, 50)
	healthy.Set(2, 1, 0.5)
	damaged.Set(2, 1, 0.25)
	ecs.AddResource(world, &res.HealthyTreeAttractionCoarse{Grid: healthy})
	ecs.AddResource(world, &res.DamagedTreeAttractionCoarse{Grid: damaged})

	for _, tc := range []struct {
		damagedTrees bool
		want         float64
	}{{false, 0.5}, {true, 0.25}} {
		o := CoarseTreeAttractionMap{DamagedTrees: tc.damagedTrees}
		o.Initialize(world)

		width, height := o.Dims()
		assert.Equal(t, 3, width)
		assert.Equal(t, 2, height)
		assert.Equal(t, 100.0, o.X(2))
		assert.Equal(t, 50.0, o.Y(1))

		values := o.Values(world)
		// Row-major: cell (2, 1) is at index 1*3+2.
		assert.InDelta(t, math.Log(tc.want), values[5], 1e-12)
		assert.True(t, math.IsInf(values[0], -1), "an empty cell's log must be -Inf")
	}
}
