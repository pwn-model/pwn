package sys

import (
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

	beetleMap := ecs.NewMap2[comp.BeetlePosition, comp.EmergenceTick](world)
	for range 50 {
		beetleMap.NewEntity(&comp.BeetlePosition{X: 100, Y: 25}, &comp.EmergenceTick{TickOfEmergence: 0})
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
	beetleMap := ecs.NewMap2[comp.BeetlePosition, comp.EmergenceTick](world)
	for range 10 {
		beetleMap.NewEntity(&comp.BeetlePosition{X: 100, Y: 25}, &comp.EmergenceTick{TickOfEmergence: -1})
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
