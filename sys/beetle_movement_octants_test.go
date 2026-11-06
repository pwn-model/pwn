package sys

import (
	"math/rand/v2"
	"testing"

	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
	"github.com/stretchr/testify/assert"
)

// ringOffsets are healthy trees around the brood tree, one per octant.
var ringOffsets = [][2]int{{-15, 0}, {15, 0}, {0, -15}, {0, 15}, {-12, -12}, {12, 12}, {-12, 12}, {12, -12}}

// beetleMovementOctantsSetup configures setupBeetleMovementOctantsWorld.
type beetleMovementOctantsSetup struct {
	offsets              [][2]int // Healthy trees, relative to the brood tree.
	n                    int      // Number of beetles.
	tickOfEmergence      int
	start                [2]int // Start of the beetles, relative to the brood tree.
	leaveTreeProbability float64
	emergenceFlight      float64
	distancePerStep      float64
}

// setupBeetleMovementOctantsWorld creates a world with healthy trees at the
// given offsets around a damaged brood tree at (100, 25), both octant
// resources, and beetles at the brood tree plus start. The current tick is
// 0. Returns the world and the movement system, initialized.
func setupBeetleMovementOctantsWorld(t *testing.T, c beetleMovementOctantsSetup) (*ecs.World, *BeetleMovementOctants) {
	t.Helper()

	world, posMap := setupTreeAttractionWorld(t)
	ecs.GetResource[resource.Rand](world).Source = res.NewXoshiro256pp(1)

	for _, o := range c.offsets {
		posMap.NewEntity(&comp.Position{X: 100 + o[0], Y: 25 + o[1]})
	}
	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	damagedMap.NewEntity(&comp.Position{X: 100, Y: 25}, &comp.Damaged{})

	healthy := TreeAttractionOctants{TickOfYear: 0, HalfDistance: 50, DensityRadius: 20, DensityWeight: 0}
	healthy.Initialize(world)
	damaged := TreeAttractionOctants{TickOfYear: 0, DamagedTrees: true, HalfDistance: 50, DensityRadius: 20, DensityWeight: 0}
	damaged.Initialize(world)
	healthy.Update(world)
	damaged.Update(world)

	n := c.n
	if n == 0 {
		n = 50
	}
	beetleMap := ecs.NewMap2[comp.BeetlePosition, comp.EmergenceTick](world)
	for range n {
		beetleMap.NewEntity(
			&comp.BeetlePosition{X: 100 + c.start[0], Y: 25 + c.start[1]},
			&comp.EmergenceTick{TickOfEmergence: c.tickOfEmergence},
		)
	}

	distance := c.distancePerStep
	if distance == 0 {
		distance = 2000
	}
	s := BeetleMovementOctants{
		StepsPerTick:            1,
		DistancePerStep:         distance,
		DurationFeeding:         3,
		FeedingProbability:      0.5,
		LeaveTreeProbability:    c.leaveTreeProbability,
		EmergenceFlightDistance: c.emergenceFlight,
	}
	s.Initialize(world)

	return world, &s
}

// octantSourcesWith returns octant sources with the given attraction
// values, and a non-zero offset for each non-empty octant.
func octantSourcesWith(attraction [8]float64) res.OctantSources {
	var r res.OctantSources
	for k, a := range attraction {
		r.Attraction[k] = float32(a)
		if a > 0 {
			r.Offsets[k] = [2]int16{int16(k + 1), 0}
		}
	}
	return r
}

func TestSelectOctantWithoutSourcesDrawsNothing(t *testing.T) {
	src := res.NewXoshiro256pp(1)
	r := res.OctantSources{}
	assert.Equal(t, -1, selectOctant(rand.New(src), &r))
	assert.Equal(t, res.NewXoshiro256pp(1).Uint64(), src.Uint64())
}

func TestSelectOctantNeverSelectsEmptyOctants(t *testing.T) {
	rng := rand.New(res.NewXoshiro256pp(1))
	r := octantSourcesWith([8]float64{0, 0, 0.5, 0, 0, 0, 0, 0})
	for range 100 {
		assert.Equal(t, 2, selectOctant(rng, &r))
	}

	r = octantSourcesWith([8]float64{0, 0.1, 0, 0, 0.3, 0, 0, 0.6})
	for range 1000 {
		assert.Contains(t, []int{1, 4, 7}, selectOctant(rng, &r))
	}
}

func TestSelectOctantIsProportionalToAttraction(t *testing.T) {
	rng := rand.New(res.NewXoshiro256pp(1))
	weights := [8]float64{0.1, 0, 0.2, 0, 0.3, 0, 0.4, 0}
	r := octantSourcesWith(weights)
	n := 100_000
	var counts [8]int
	for range n {
		counts[selectOctant(rng, &r)]++
	}
	for k := range 8 {
		assert.InDelta(t, weights[k], float64(counts[k])/float64(n), 0.01)
	}
}

// Unlike BeetleMovement, beetles from the same cell do not all fly to the
// same tree: each of the 8 ring trees is in its own octant.
func TestBeetleMovementOctantsNoEmergenceFlightSpreadTargets(t *testing.T) {
	world, s := setupBeetleMovementOctantsWorld(t, beetleMovementOctantsSetup{
		offsets: ringOffsets, leaveTreeProbability: 0.25,
	})
	s.Update(world)

	counts := beetlePositions(world)
	presence := ecs.GetResource[res.HealthyTreeAttractionOctants](world).Presence
	total := 0
	for c, n := range counts {
		total += n
		assert.True(t, presence.Get(c[0], c[1]), "cell %v", c)
	}
	assert.Equal(t, 50, total)
	assert.Greater(t, len(counts), 3)
}

// Two trees in opposite octants, 15 and 30 cells away. At a half distance
// of 50 m (5 cells), the nearer one is 2^3 = 8 times as attractive, so it
// gets 8/9 of the beetles.
func TestBeetleMovementOctantsTargetsByAttraction(t *testing.T) {
	n := 9000
	world, s := setupBeetleMovementOctantsWorld(t, beetleMovementOctantsSetup{
		offsets: [][2]int{{15, 0}, {-30, 0}}, n: n, leaveTreeProbability: 0.25,
	})
	s.Update(world)

	counts := beetlePositions(world)
	assert.Equal(t, n, counts[[2]int{115, 25}]+counts[[2]int{70, 25}])
	assert.InDelta(t, 8.0/9.0, float64(counts[[2]int{115, 25}])/float64(n), 0.015)
	assert.InDelta(t, 1.0/9.0, float64(counts[[2]int{70, 25}])/float64(n), 0.015)
}

func TestBeetleMovementOctantsEmergenceFlightOnlyOnce(t *testing.T) {
	world, s := setupBeetleMovementOctantsWorld(t, beetleMovementOctantsSetup{
		offsets: ringOffsets, emergenceFlight: 300,
	})
	s.Update(world)
	before := beetlePositions(world)

	ecs.GetResource[res.Time](world).Tick = 1
	s.Update(world)

	assert.Equal(t, before, beetlePositions(world))
}

func TestBeetleMovementOctantsEmergenceFlightOutOfWorldDies(t *testing.T) {
	world, s := setupBeetleMovementOctantsWorld(t, beetleMovementOctantsSetup{
		offsets: ringOffsets, leaveTreeProbability: 0.25, emergenceFlight: 2000,
	})
	s.Update(world)

	// From (100, 25) in a 200 x 50 cell world, the flight of up to 200 cells
	// leaves the world in most directions and distances.
	counts := beetlePositions(world)
	presence := ecs.GetResource[res.HealthyTreeAttractionOctants](world).Presence
	total := 0
	for c, n := range counts {
		total += n
		assert.True(t, presence.Contains(c[0], c[1]), "cell %v", c)
	}
	assert.Greater(t, total, 0)
	assert.Less(t, total, 50)
}

// Beetles on the ring tree at (115, 25), emerged in an earlier tick (no
// emergence flight), all leaving: they fly directly to other ring trees,
// without a walk phase, and none leaves the world.
func TestBeetleMovementOctantsLeavingTreeFliesToAnotherTree(t *testing.T) {
	world, s := setupBeetleMovementOctantsWorld(t, beetleMovementOctantsSetup{
		offsets: ringOffsets, tickOfEmergence: -1, start: [2]int{15, 0}, leaveTreeProbability: 1,
	})
	s.Update(world)

	counts := beetlePositions(world)
	presence := ecs.GetResource[res.HealthyTreeAttractionOctants](world).Presence
	total := 0
	for c, n := range counts {
		total += n
		assert.True(t, presence.Get(c[0], c[1]), "cell %v", c)
	}
	assert.Equal(t, 50, total)
	assert.NotContains(t, counts, [2]int{115, 25})
}

func TestBeetleMovementOctantsStayingOnATree(t *testing.T) {
	world, s := setupBeetleMovementOctantsWorld(t, beetleMovementOctantsSetup{
		offsets: ringOffsets, tickOfEmergence: -1, start: [2]int{15, 0},
	})
	s.Update(world)

	assert.Equal(t, map[[2]int]int{{115, 25}: 50}, beetlePositions(world))
	feeding := ecs.GetResource[res.FeedingInfectedBeetles](world)
	assert.Equal(t, uint32(50), feeding.Get(115, 25))
}

// No healthy trees at all; beetles emerged in an earlier tick (no emergence
// flight) and still feeding (no feeding draw), so they neither move nor
// draw any random number.
func TestBeetleMovementOctantsWithoutVisibleTreesStays(t *testing.T) {
	world, s := setupBeetleMovementOctantsWorld(t, beetleMovementOctantsSetup{
		tickOfEmergence: -1, start: [2]int{30, 10}, leaveTreeProbability: 0.25,
	})
	s.Update(world)

	assert.Equal(t, map[[2]int]int{{130, 35}: 50}, beetlePositions(world))
	src := ecs.GetResource[resource.Rand](world).Source
	assert.Equal(t, res.NewXoshiro256pp(1).Uint64(), src.Uint64())
}

// A single tree 30 cells away, but only 10 cells of flight per step: the
// beetles get 10 cells closer per step and reach it in the third step.
func TestBeetleMovementOctantsPartialFlight(t *testing.T) {
	world, s := setupBeetleMovementOctantsWorld(t, beetleMovementOctantsSetup{
		offsets: [][2]int{{30, 0}}, tickOfEmergence: -1, distancePerStep: 100,
	})
	for _, expected := range []int{110, 120, 130} {
		s.Update(world)
		assert.Equal(t, map[[2]int]int{{expected, 25}: 50}, beetlePositions(world))
	}
}
