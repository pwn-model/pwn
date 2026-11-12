package sys

import (
	"math"
	"testing"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

// setupBeetleMovementWorld creates a world with a ring of healthy trees
// around a damaged brood tree at (100, 25), both attraction fields, and
// 50 beetles emerging from the brood tree in the current tick. Returns the
// world and the movement system, initialized.
func setupBeetleMovementWorld(t *testing.T, emergenceFlightDistance float64) (*ecs.World, *BeetleMovement) {
	t.Helper()

	world, posMap := setupTreeAttractionWorld(t)
	ecs.GetResource[resource.Rand](world).Source = res.NewXoshiro256pp(1)

	for _, o := range [][2]int{{-15, 0}, {15, 0}, {0, -15}, {0, 15}, {-12, -12}, {12, 12}, {-12, 12}, {12, -12}} {
		posMap.NewEntity(&comp.Position{X: 100 + o[0], Y: 25 + o[1]})
	}
	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	damagedMap.NewEntity(&comp.Position{X: 100, Y: 25}, &comp.Damaged{})

	healthy := TreeAttraction{TickOfYear: 0, HalfDistance: 50, DensityRadius: 20, DensityWeight: 0}
	healthy.Initialize(world)
	damaged := TreeAttraction{TickOfYear: 0, DamagedTrees: true, HalfDistance: 50, DensityRadius: 20, DensityWeight: 0}
	damaged.Initialize(world)
	healthy.Update(world)
	damaged.Update(world)

	beetleMap := ecs.NewMap3[comp.BeetlePosition, comp.EmergenceTick, comp.WalkDirection](world)
	for range 50 {
		beetleMap.NewEntity(&comp.BeetlePosition{X: 100, Y: 25}, &comp.EmergenceTick{TickOfEmergence: 0}, &comp.WalkDirection{})
	}

	s := BeetleMovement{
		StepsPerTick:            1,
		DistancePerStep:         2000,
		DurationFeeding:         3,
		FeedingProbability:      0.5,
		LeaveTreeProbability:    0.25,
		EmergenceFlightDistance: emergenceFlightDistance,
	}
	s.Initialize(world)

	return world, &s
}

// beetlePositions returns the number of beetles per cell.
func beetlePositions(world *ecs.World) map[[2]int]int {
	counts := map[[2]int]int{}
	q := ecs.NewFilter1[comp.BeetlePosition](world).Query()
	for q.Next() {
		p := q.Get()
		counts[[2]int{p.X, p.Y}]++
	}
	return counts
}

// Without an emergence flight, all beetles from the same tree fly to the
// same first target.
func TestBeetleMovementNoEmergenceFlightSameTarget(t *testing.T) {
	world, s := setupBeetleMovementWorld(t, 0)
	s.Update(world)

	assert.Len(t, beetlePositions(world), 1)
}

// With an emergence flight, beetles from the same tree spread out over
// several first targets, all of them healthy trees.
func TestBeetleMovementEmergenceFlightSpreadsTargets(t *testing.T) {
	world, s := setupBeetleMovementWorld(t, 300)
	s.Update(world)

	counts := beetlePositions(world)
	assert.Greater(t, len(counts), 3)

	presence := ecs.GetResource[res.HealthyTreeAttraction](world).Presence
	for c := range counts {
		assert.True(t, presence.Get(c[0], c[1]), "beetle at %v is not on a healthy tree", c)
	}
}

// The emergence flight only happens in the beetles' first step: in later
// ticks, beetles on a tree stay there unless they leave it.
func TestBeetleMovementEmergenceFlightOnlyOnce(t *testing.T) {
	world, s := setupBeetleMovementWorld(t, 300)
	s.LeaveTreeProbability = 0
	s.Update(world)
	before := beetlePositions(world)

	ecs.GetResource[res.Time](world).Tick = 1
	s.Update(world)

	assert.Equal(t, before, beetlePositions(world))
}

// Beetles whose emergence flight leaves the world die.
func TestBeetleMovementEmergenceFlightOutOfWorldDies(t *testing.T) {
	world, s := setupBeetleMovementWorld(t, 2000)
	s.Update(world)

	// From (100, 25) in a 200 x 50 cell world, the flight of up to 200
	// cells leaves the world in most directions and distances.
	counts := beetlePositions(world)
	total := 0
	presence := ecs.GetResource[res.HealthyTreeAttraction](world).Presence
	for c, n := range counts {
		total += n
		assert.True(t, presence.Contains(c[0], c[1]), "beetle at %v is outside the world", c)
	}
	assert.Less(t, total, 50)
	assert.Positive(t, total)
}

// A beetle leaving the only tree of its target type walks until it leaves
// the world (as all cells have that tree as source), and dies.
func TestBeetleMovementLeaveTreeOutOfWorldDies(t *testing.T) {
	world, posMap := setupTreeAttractionWorld(t)
	ecs.GetResource[resource.Rand](world).Source = res.NewXoshiro256pp(1)
	posMap.NewEntity(&comp.Position{X: 100, Y: 25})

	healthy := TreeAttraction{TickOfYear: 0, HalfDistance: 50, DensityRadius: 20, DensityWeight: 0}
	healthy.Initialize(world)
	damaged := TreeAttraction{TickOfYear: 0, DamagedTrees: true, HalfDistance: 50, DensityRadius: 20, DensityWeight: 0}
	damaged.Initialize(world)
	healthy.Update(world)
	damaged.Update(world)

	// Emerged in an earlier tick, so no emergence flight.
	beetleMap := ecs.NewMap3[comp.BeetlePosition, comp.EmergenceTick, comp.WalkDirection](world)
	for range 10 {
		beetleMap.NewEntity(&comp.BeetlePosition{X: 100, Y: 25}, &comp.EmergenceTick{TickOfEmergence: -1}, &comp.WalkDirection{})
	}

	s := BeetleMovement{
		StepsPerTick:            1,
		DistancePerStep:         2000,
		DurationFeeding:         3,
		FeedingProbability:      0.5,
		LeaveTreeProbability:    1,
		EmergenceFlightDistance: 0,
	}
	s.Initialize(world)
	s.Update(world)

	assert.Empty(t, beetlePositions(world))
}

// Without an emergence flight, no random numbers are drawn for it: emerging
// beetles still feeding (no feeding draw), not on a healthy tree (no leave
// draw), fly to their target without using the RNG at all.
func TestBeetleMovementNoEmergenceFlightDrawsNoRandomNumbers(t *testing.T) {
	world, s := setupBeetleMovementWorld(t, 0)
	s.Update(world)

	src := ecs.GetResource[resource.Rand](world).Source
	assert.Equal(t, res.NewXoshiro256pp(1).Uint64(), src.Uint64())
}

// setupWalkWorld creates a world with healthy trees at the given offsets
// around a damaged tree at (100, 25), both attraction fields (computed
// without density weighting), and n beetles at (x, y) that emerged at
// tickOfEmergence. The current tick is 0. Returns the world and the movement
// system with 5 cells (50 m) of budget per step, initialized.
func setupWalkWorld(t *testing.T, offsets [][2]int, n int, tickOfEmergence int, x, y int, leaveTreeProbability float64) (*ecs.World, *BeetleMovement) {
	t.Helper()

	world, posMap := setupTreeAttractionWorld(t)
	ecs.GetResource[resource.Rand](world).Source = res.NewXoshiro256pp(1)
	for _, o := range offsets {
		posMap.NewEntity(&comp.Position{X: 100 + o[0], Y: 25 + o[1]})
	}
	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	damagedMap.NewEntity(&comp.Position{X: 100, Y: 25}, &comp.Damaged{})

	healthy := TreeAttraction{TickOfYear: 0, HalfDistance: 50, DensityRadius: 20, DensityWeight: 0}
	healthy.Initialize(world)
	damaged := TreeAttraction{TickOfYear: 0, DamagedTrees: true, HalfDistance: 50, DensityRadius: 20, DensityWeight: 0}
	damaged.Initialize(world)
	healthy.Update(world)
	damaged.Update(world)

	beetleMap := ecs.NewMap3[comp.BeetlePosition, comp.EmergenceTick, comp.WalkDirection](world)
	for range n {
		beetleMap.NewEntity(&comp.BeetlePosition{X: x, Y: y}, &comp.EmergenceTick{TickOfEmergence: tickOfEmergence}, &comp.WalkDirection{})
	}

	s := BeetleMovement{
		StepsPerTick:            1,
		DistancePerStep:         50,
		DurationFeeding:         3,
		FeedingProbability:      0.5,
		LeaveTreeProbability:    leaveTreeProbability,
		EmergenceFlightDistance: 0,
	}
	s.Initialize(world)

	return world, &s
}

// beetleWalk is the position and walk direction of a beetle.
type beetleWalk struct {
	Pos comp.BeetlePosition
	Dir comp.WalkDirection
}

// beetleWalks returns the positions and walk directions of all beetles.
func beetleWalks(world *ecs.World) []beetleWalk {
	walks := []beetleWalk{}
	q := ecs.NewFilter2[comp.BeetlePosition, comp.WalkDirection](world).Query()
	for q.Next() {
		p, d := q.Get()
		walks = append(walks, beetleWalk{*p, *d})
	}
	return walks
}

// isWalking reports whether a beetle walks away from a tree.
func isWalking(d comp.WalkDirection) bool {
	return d.Dx != 0 || d.Dy != 0
}

// rngState returns a copy of the world's random number generator state.
func rngState(world *ecs.World) res.Xoshiro256pp {
	return *ecs.GetResource[resource.Rand](world).Source.(*res.Xoshiro256pp)
}

// A beetle leaving the only tree (whose area is the whole world) with a
// budget of 5 cells per step runs out of budget before leaving the area. It
// stores its direction and mode, and continues in the next step without
// drawing any random numbers, instead of flying back to the tree.
func TestBeetleMovementWalkContinuesInStoredDirection(t *testing.T) {
	world, s := setupWalkWorld(t, [][2]int{{0, 0}}, 20, -1, 100, 25, 1)

	s.Update(world)
	first := beetleWalks(world)
	assert.Len(t, first, 20)
	for _, w := range first {
		assert.True(t, isWalking(w.Dir))
		assert.True(t, w.Dir.OnTheWay)
		assert.True(t, w.Dir.Feeding)
		d := math.Hypot(float64(w.Pos.X-100), float64(w.Pos.Y-25))
		assert.True(t, d >= 4 && d <= 6, "distance %f after first step", d)
	}

	expected := rngState(world)
	s.Update(world)
	assert.Equal(t, expected, rngState(world))

	second := beetleWalks(world)
	assert.Len(t, second, 20)
	for i, w := range second {
		w1 := first[i]
		assert.Equal(t, w1.Dir, w.Dir)
		// Each step rounds to cells, off by up to ~0.7 cells.
		d := math.Hypot(float64(w.Pos.X-100), float64(w.Pos.Y-25))
		assert.True(t, d >= 8.5 && d <= 11.5, "distance %f after second step", d)
		// Still moving away from the tree, in the stored direction.
		dot := float64(w.Pos.X-w1.Pos.X)*float64(w1.Dir.Dx) + float64(w.Pos.Y-w1.Pos.Y)*float64(w1.Dir.Dy)
		assert.Greater(t, dot, 4.0)
	}
}

// With two trees 30 cells apart, beetles leaving the west tree towards the
// east leave its area after 15 cells, i.e. in their third step, and then fly
// to the east tree. Beetles on a tree never have a stored direction.
func TestBeetleMovementWalkDirectionClearedWhenLeavingArea(t *testing.T) {
	world, s := setupWalkWorld(t, [][2]int{{0, 0}, {30, 0}}, 50, -1, 100, 25, 1)
	presence := ecs.GetResource[res.HealthyTreeAttraction](world).Presence

	reachedEast := false
	for range 10 {
		s.Update(world)
		for _, w := range beetleWalks(world) {
			onTree := presence.Get(w.Pos.X, w.Pos.Y)
			assert.False(t, onTree && isWalking(w.Dir), "beetle on a tree at %v is walking", w.Pos)
			if w.Pos.X == 130 && w.Pos.Y == 25 {
				reachedEast = true
			}
		}
	}
	assert.True(t, reachedEast)
}

// Beetles in the egg laying phase between a healthy (east) and a damaged
// (west) tree, 30 cells from each, with a budget of 5 cells per step, fly
// towards one of them by mode. They keep their mode while on the way,
// without drawing it anew, until they arrive.
func TestBeetleMovementKeepsModeWhileFlyingToTarget(t *testing.T) {
	world, s := setupWalkWorld(t, [][2]int{{60, 0}}, 20, -10, 130, 25, 0)

	target := func(feeding bool, east, west int) comp.BeetlePosition {
		if feeding {
			return comp.BeetlePosition{X: east, Y: 25}
		}
		return comp.BeetlePosition{X: west, Y: 25}
	}

	s.Update(world)
	first := beetleWalks(world)
	modes := map[bool]bool{}
	for _, w := range first {
		assert.True(t, w.Dir.OnTheWay)
		assert.False(t, isWalking(w.Dir))
		assert.Equal(t, target(w.Dir.Feeding, 135, 125), w.Pos)
		modes[w.Dir.Feeding] = true
	}
	// Both modes occur.
	assert.Len(t, modes, 2)

	expected := rngState(world)
	for range 4 {
		s.Update(world)
	}
	assert.Equal(t, expected, rngState(world))
	for i, w := range beetleWalks(world) {
		assert.True(t, w.Dir.OnTheWay)
		assert.Equal(t, first[i].Dir.Feeding, w.Dir.Feeding)
		assert.Equal(t, target(first[i].Dir.Feeding, 155, 105), w.Pos)
	}

	// Arrived after 6 steps, no longer on the way.
	s.Update(world)
	for i, w := range beetleWalks(world) {
		assert.Equal(t, comp.WalkDirection{}, w.Dir)
		assert.Equal(t, target(first[i].Dir.Feeding, 160, 100), w.Pos)
	}
}
