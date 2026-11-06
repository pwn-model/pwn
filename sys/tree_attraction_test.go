package sys

import (
	"math"
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/util"
	"github.com/stretchr/testify/assert"
)

func setupTreeAttractionWorld(t *testing.T) (*ecs.World, *ecs.Map1[comp.Position]) {
	t.Helper()

	a := app.New()
	ws := res.NewWorldSize(2000, 500, 10)
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
	grid := ecs.GetResource[res.HealthyTreeAttraction](world).Attraction
	// A single isolated tree seeds at count^0=1; 10 cells away (a purely
	// orthogonal offset, so the chamfer sweep is exact here, not
	// approximate), the value should be exactly decay^10 -- confirming
	// util.PropagateAttraction decays multiplicatively rather than
	// subtracting a fixed cost per step.
	assert.InDelta(t, math.Pow(decay, 10), grid.Get(60, 25), 1e-9)
}

// Two independently configured instances -- one for healthy trees, one for
// damaged -- can be run in the same world side by side (as they would be
// added to a real scheduler twice), each driven only by its own tree type
// and its own HalfDistance, without interfering with each other.
func TestTreeAttractionHealthyAndDamagedInstancesAreIndependent(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	damagedMap.NewEntity(&comp.Position{X: 80, Y: 25}, &comp.Damaged{})

	healthyHalfDistance := 50 * math.Ln2
	healthy := TreeAttraction{TickOfYear: 0, HalfDistance: healthyHalfDistance, DensityRadius: 20, DensityWeight: 0}
	healthy.Initialize(world)

	damagedHalfDistance := 200 * math.Ln2
	damaged := TreeAttraction{
		TickOfYear: 0, DamagedTrees: true, HalfDistance: damagedHalfDistance, DensityRadius: 20, DensityWeight: 0,
	}
	damaged.Initialize(world)

	healthy.Update(world)
	damaged.Update(world)

	healthyGrid := ecs.GetResource[res.HealthyTreeAttraction](world).Attraction
	damagedGrid := ecs.GetResource[res.DamagedTreeAttraction](world).Attraction

	// The healthy field is driven only by the healthy tree at (50, 25), 10
	// cells from the query point, using the healthy instance's own decay.
	healthyDecay := math.Exp(-10.0 * math.Ln2 / healthyHalfDistance)
	assert.InDelta(t, math.Pow(healthyDecay, 10), healthyGrid.Get(60, 25), 1e-9)

	// The damaged field is driven only by the damaged tree at (80, 25), 20
	// cells from the same query point, using the damaged instance's own
	// (different) decay -- confirming the two run independently.
	damagedDecay := math.Exp(-10.0 * math.Ln2 / damagedHalfDistance)
	assert.InDelta(t, math.Pow(damagedDecay, 20), damagedGrid.Get(60, 25), 1e-9)
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
	grid := ecs.GetResource[res.HealthyTreeAttraction](world).Attraction
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
	grid := ecs.GetResource[res.HealthyTreeAttraction](world).Attraction
	decay := math.Exp(-10.0 * math.Ln2 / halfDistance)
	assert.InDelta(t, math.Pow(decay, 10), grid.Get(60, 25), 1e-9)

	source := ecs.GetResource[res.HealthyTreeAttraction](world).Source
	assert.Equal(t, source.Index(50, 25), source.Get(60, 25))
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
	grid := ecs.GetResource[res.HealthyTreeAttraction](world).Attraction
	decay := math.Exp(-10.0 * math.Ln2 / halfDistance)
	maxCount := (2.0*2.0 + 1.0) * (2.0*2.0 + 1.0) // DensityRadius=20, CellSize=10 -> densityRadiusCells=2.
	isolatedValue := (1.0 / maxCount) * math.Pow(decay, 10)
	clusterValue := (9.0 / maxCount) * math.Pow(decay, 39)
	assert.Greater(t, clusterValue, isolatedValue, "sanity check: chosen HalfDistance must make the cluster the stronger source")
	assert.InDelta(t, clusterValue, grid.Get(60, 25), 1e-9)

	// The source is the cluster's nearest tree, not the closer lone tree.
	source := ecs.GetResource[res.HealthyTreeAttraction](world).Source
	assert.Equal(t, source.Index(99, 25), source.Get(60, 25))
}

// A fully-occupied patch (a tree in every cell of its own DensityRadius
// window) has an occupancy fraction of exactly 1, so its seed is exactly
// 1 regardless of DensityWeight -- and since util.PropagateAttraction's
// max-relaxation can never exceed the largest seed anywhere in the grid, no
// cell of the resulting field can exceed 1 either, however densely
// populated the world is or how high DensityWeight is set.
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

	grid := ecs.GetResource[res.HealthyTreeAttraction](world).Attraction
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

	grid := ecs.GetResource[res.HealthyTreeAttraction](world).Attraction
	assert.InDelta(t, 1.0, grid.Get(50, 25), 1e-9)
}

func TestTreeAttractionSkipsWrongTickOfYear(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	s := TreeAttraction{TickOfYear: 5, HalfDistance: 200 * math.Ln2, DensityRadius: 20, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	grid := ecs.GetResource[res.HealthyTreeAttraction](world).Attraction
	assert.Equal(t, 0.0, grid.Get(50, 25))
}

// Presence is updated together with the attraction field, only on
// TickOfYear, and only for the instance's own tree type.
func TestTreeAttractionPresenceOnTickOfYear(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	e := posMap.NewEntity(&comp.Position{X: 50, Y: 25})

	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	damagedMap.NewEntity(&comp.Position{X: 80, Y: 25}, &comp.Damaged{})

	healthy := TreeAttraction{TickOfYear: 0, HalfDistance: 50, DensityRadius: 20, DensityWeight: 1}
	healthy.Initialize(world)
	damaged := TreeAttraction{TickOfYear: 0, DamagedTrees: true, HalfDistance: 50, DensityRadius: 20, DensityWeight: 1}
	damaged.Initialize(world)

	healthy.Update(world)
	damaged.Update(world)

	healthyPresence := ecs.GetResource[res.HealthyTreeAttraction](world).Presence
	damagedPresence := ecs.GetResource[res.DamagedTreeAttraction](world).Presence
	assert.True(t, healthyPresence.Get(50, 25))
	assert.False(t, healthyPresence.Get(80, 25))
	assert.True(t, damagedPresence.Get(80, 25))
	assert.False(t, damagedPresence.Get(50, 25))

	// A tree becoming damaged is not reflected outside TickOfYear...
	ecs.NewMap[comp.Damaged](world).Add(e, &comp.Damaged{})
	timeRes := ecs.GetResource[res.Time](world)
	timeRes.TickOfYear = 1
	healthy.Update(world)
	damaged.Update(world)

	assert.True(t, healthyPresence.Get(50, 25))
	assert.False(t, damagedPresence.Get(50, 25))

	// ...but on the next TickOfYear.
	timeRes.TickOfYear = 0
	healthy.Update(world)
	damaged.Update(world)

	assert.False(t, healthyPresence.Get(50, 25))
	assert.True(t, damagedPresence.Get(50, 25))
}

// The seeds computed by fillFromQuery, with the summed-area table and the
// seeding fused into one pass, match a brute-force window count, including
// near the world's edges.
func TestTreeAttractionSeedsMatchBruteForce(t *testing.T) {
	a := app.New()
	ws := res.NewWorldSize(300, 200, 10)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 0})
	posMap := ecs.NewMap1[comp.Position](a.World)

	src := res.NewXoshiro256pp(1)
	w, h := ws.Width(), ws.Height()
	for range 200 {
		posMap.NewEntity(&comp.Position{X: int(src.Uint64() % uint64(w)), Y: int(src.Uint64() % uint64(h))})
	}

	s := TreeAttraction{TickOfYear: 0, HalfDistance: 50, DensityRadius: 40, DensityWeight: 1.5}
	s.Initialize(a.World)
	s.fillFromQuery()

	r := s.density.RadiusCells()
	for x := range w {
		for y := range h {
			if !s.presence.Get(x, y) {
				assert.Equal(t, 0.0, s.attraction.Get(x, y))
				continue
			}
			count := 0
			for xx := max(x-r, 0); xx <= min(x+r, w-1); xx++ {
				for yy := max(y-r, 0); yy <= min(y+r, h-1); yy++ {
					if s.presence.Get(xx, yy) {
						count++
					}
				}
			}
			expected := math.Pow(float64(count)/s.density.MaxCount(), s.DensityWeight)
			assert.Equal(t, expected, s.attraction.Get(x, y), "cell (%d, %d)", x, y)
		}
	}
}

// Without any matching tree, no cell has a source.
func TestTreeAttractionNoTreesNoSource(t *testing.T) {
	world, _ := setupTreeAttractionWorld(t)

	s := TreeAttraction{TickOfYear: 0, HalfDistance: 50, DensityRadius: 20, DensityWeight: 1}
	s.Initialize(world)
	s.Update(world)

	source := ecs.GetResource[res.HealthyTreeAttraction](world).Source
	for x := range source.Width() {
		for y := range source.Height() {
			assert.Equal(t, res.NoCell, source.Get(x, y))
		}
	}
}

// Every cell's traced source is a tree cell, the cell's value is exactly
// that source's seed decayed over the octile distance between them, and no
// other tree contributes more -- i.e. the source is the tree that wins the
// max-relaxation, checked by brute force over all trees.
func TestTreeAttractionSourceMatchesBruteForce(t *testing.T) {
	a := app.New()
	ws := res.NewWorldSize(300, 200, 10)
	ecs.AddResource(a.World, &ws)
	ecs.AddResource(a.World, &res.Time{TickOfYear: 0})
	posMap := ecs.NewMap1[comp.Position](a.World)

	src := res.NewXoshiro256pp(1)
	w, h := ws.Width(), ws.Height()
	for range 40 {
		posMap.NewEntity(&comp.Position{X: int(src.Uint64() % uint64(w)), Y: int(src.Uint64() % uint64(h))})
	}

	s := TreeAttraction{TickOfYear: 0, HalfDistance: 50, DensityRadius: 40, DensityWeight: 1.5}
	s.Initialize(a.World)
	s.fillFromQuery()

	seeds := res.NewGrid[float64](w, h, ws.CellSize())
	var trees [][2]int
	for x := range w {
		for y := range h {
			seeds.Set(x, y, s.attraction.Get(x, y))
			if s.presence.Get(x, y) {
				trees = append(trees, [2]int{x, y})
			}
		}
	}

	util.PropagateAttraction(&s.attraction, &s.source, s.HalfDistance)

	decay := math.Exp(-float64(ws.CellSize()) * math.Ln2 / s.HalfDistance)
	contribution := func(x, y int, c [2]int) float64 {
		dx, dy := math.Abs(float64(x-c[0])), math.Abs(float64(y-c[1]))
		dist := max(dx, dy) + (math.Sqrt2-1)*min(dx, dy)
		return seeds.Get(c[0], c[1]) * math.Pow(decay, dist)
	}

	for x := range w {
		for y := range h {
			cx, cy := s.source.Coords(s.source.Get(x, y))
			c := [2]int{cx, cy}
			if !assert.True(t, s.presence.Get(cx, cy), "cell (%d, %d): source %v is not a tree", x, y, c) {
				continue
			}
			v := s.attraction.Get(x, y)
			assert.InEpsilon(t, contribution(x, y, c), v, 1e-9, "cell (%d, %d)", x, y)
			for _, tree := range trees {
				assert.LessOrEqual(t, contribution(x, y, tree), v*(1+1e-9), "cell (%d, %d), tree %v", x, y, tree)
			}
		}
	}
}
