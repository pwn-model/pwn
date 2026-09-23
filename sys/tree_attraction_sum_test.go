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

func peakOf(grid *res.Grid[float64]) float64 {
	max := 0.0
	for x := range grid.Width() {
		for y := range grid.Height() {
			if v := grid.Get(x, y); v > max {
				max = v
			}
		}
	}
	return max
}

func runTreeAttractionSum(t *testing.T, width, height, scale int) float64 {
	t.Helper()

	a := app.New()
	ws := res.NewWorldSize(width, height, 10, 100)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 0})
	posMap := ecs.NewMap1[comp.Position](a.World)
	posMap.NewEntity(&comp.Position{X: (width / 10) / 2, Y: (height / 10) / 2})

	s := TreeAttractionSum{TickOfYear: 0, Scale: scale}
	s.Initialize(a.World)
	s.Update(a.World)

	return peakOf(&ecs.GetResource[res.HealthyTreeAttraction](a.World).Grid)
}

func TestTreeAttractionSumSkipsWrongTickOfYear(t *testing.T) {
	a := app.New()
	ws := res.NewWorldSize(200, 200, 10, 100)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 3})
	posMap := ecs.NewMap1[comp.Position](a.World)
	posMap.NewEntity(&comp.Position{X: 10, Y: 10})

	s := TreeAttractionSum{TickOfYear: 0, Scale: 5}
	s.Initialize(a.World)
	s.Update(a.World)

	grid := ecs.GetResource[res.HealthyTreeAttraction](a.World)
	assert.Equal(t, 0.0, grid.Get(10, 10))
}

// At a decay length well under the world's base cell size (Scale=5,
// CellSize=10, decay=exp(-2)~=0.135, comfortably under fillGrid's ~0.25
// stability threshold of 4*decay<1), the field is stable: the peak value
// converges to a fixed point independent of how big the surrounding grid
// is.
func TestTreeAttractionSumIsStableForSmallScale(t *testing.T) {
	small := runTreeAttractionSum(t, 200, 200, 5)
	large := runTreeAttractionSum(t, 3200, 3200, 5)

	assert.InDelta(t, small, large, 1e-6,
		"a stable (small-Scale) field's peak must not depend on grid size")
}

// At a decay length comparable to or larger than the cell size -- the
// range TreeAttraction's other alternatives actually need to be useful, and
// exactly the value used in this project's config.yaml (Scale=50,
// CellSize=10) -- fillGrid's sum/multiplicative-decay combinator is
// unstable: each forward step gathers decay-weighted contributions from 4
// neighbours, and once 4*decay exceeds 1 (i.e. Scale exceeds roughly
// CellSize/ln(4) =~ 0.72*CellSize) the peak grows without bound as the
// grid grows, rather than converging to a finite density field. This test
// pins that down as a genuine divergence, not just "large values": doubling
// the grid repeatedly must keep increasing the peak, and does so fast
// enough to overflow float64 well within realistic world sizes.
func TestTreeAttractionSumDivergesForRealisticScale(t *testing.T) {
	sizes := []int{200, 400, 800, 1600}
	scale := 50 // matches config.yaml's radius/scale choice for this system family.

	var prev float64
	for i, size := range sizes {
		peak := runTreeAttractionSum(t, size, size, scale)
		if i > 0 {
			assert.Greater(t, peak, prev*10,
				"peak must keep growing sharply with grid size once decay exceeds the stability threshold")
		}
		prev = peak
	}

	// A world the size of this project's own config.yaml (4000x3000m,
	// cell_size 10 -> 400x300 cells) already overflows float64 at this
	// Scale.
	peak := runTreeAttractionSum(t, 4000, 3000, scale)
	assert.True(t, math.IsInf(peak, 1), "expected float64 overflow at a realistic world size and Scale")
}
