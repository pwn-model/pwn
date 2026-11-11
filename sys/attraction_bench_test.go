package sys

import (
	"math/rand/v2"
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/comp"
	"github.com/pwn-model/pwn/res"
)

// Benchmarks of tree attraction (TreeAttraction) and the beetle movement
// using it (BeetleMovement), on a 10 km x 10 km world with 10 m cells and a
// tree in 90 % of the cells, 3 % of them damaged.
//
// Attraction benchmarks measure one calculation of the healthy-tree field.
// Movement benchmarks measure one tick of 10,000 beetles with 7 steps each,
// without the attraction calculation, which is done once beforehand; they
// also report the time per beetle step (ns/step).
//
// Mirrors the sibling Julia implementation's benchmark/internal/bench_attraction.jl.

const (
	attractionBenchCells = 1000
	movementBenchBeetles = 10_000
	movementBenchSteps   = 7
)

// newAttractionBenchWorld creates a world with randomly placed trees, and the
// resources the attraction and movement systems need. The current tick and
// tick of year are 0.
func newAttractionBenchWorld() *ecs.World {
	a := app.New()
	world := a.World
	ws := res.NewWorldSize(10*attractionBenchCells, 10*attractionBenchCells, 10)
	ecs.AddResource(world, &ws)
	ecs.AddResource(world, &res.Time{})
	ecs.GetResource[resource.Rand](world).Source = res.NewXoshiro256pp(1)

	healthyMap := ecs.NewMap1[comp.Position](world)
	damagedMap := ecs.NewMap2[comp.Position, comp.Damaged](world)
	rng := rand.New(res.NewXoshiro256pp(1))
	for x := range attractionBenchCells {
		for y := range attractionBenchCells {
			if rng.Float64() >= 0.9 {
				continue
			}
			if rng.Float64() < 0.03 {
				damagedMap.NewEntity(&comp.Position{X: x, Y: y}, &comp.Damaged{})
			} else {
				healthyMap.NewEntity(&comp.Position{X: x, Y: y})
			}
		}
	}
	return world
}

func BenchmarkTreeAttraction(b *testing.B) {
	world := newAttractionBenchWorld()
	s := TreeAttraction{TickOfYear: 0, HalfDistance: 50, DensityRadius: 20, DensityWeight: 1}
	s.Initialize(world)

	for b.Loop() {
		s.Update(world)
	}
}

// benchmarkMovement benchmarks a movement system, with attraction computed
// once by the given healthy and damaged attraction systems. Before each
// iteration (not measured), all beetles are replaced by fresh ones at random
// positions, emerged before the current tick (no emergence flight), and the
// RNG is re-seeded.
func benchmarkMovement(b *testing.B, healthy, damaged, movement app.System) {
	world := newAttractionBenchWorld()
	for _, s := range []app.System{healthy, damaged} {
		s.Initialize(world)
		s.Update(world)
	}
	movement.Initialize(world)

	randRes := ecs.GetResource[resource.Rand](world)
	beetles := ecs.NewFilter1[comp.BeetlePosition](world)
	beetleMap := ecs.NewMap2[comp.BeetlePosition, comp.EmergenceTick](world)

	for b.Loop() {
		b.StopTimer()
		world.RemoveEntities(beetles.Batch(), nil)
		randRes.Source = res.NewXoshiro256pp(1)
		rng := rand.New(res.NewXoshiro256pp(2))
		beetleMap.NewBatchFn(movementBenchBeetles, func(_ ecs.Entity, pos *comp.BeetlePosition, et *comp.EmergenceTick) {
			pos.X, pos.Y = rng.IntN(attractionBenchCells), rng.IntN(attractionBenchCells)
			et.TickOfEmergence = -1
		})
		b.StartTimer()

		movement.Update(world)
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N*movementBenchBeetles*movementBenchSteps), "ns/step")
}

// DurationFeeding 0, so that beetles target healthy and damaged trees with
// equal probability in every step.
func BenchmarkBeetleMovement(b *testing.B) {
	benchmarkMovement(b,
		&TreeAttraction{TickOfYear: 0, HalfDistance: 50, DensityRadius: 20, DensityWeight: 1},
		&TreeAttraction{TickOfYear: 0, DamagedTrees: true, HalfDistance: 50, DensityRadius: 20, DensityWeight: 1},
		&BeetleMovement{
			StepsPerTick: movementBenchSteps, DistancePerStep: 2000, EmergenceFlightDistance: 100,
			DurationFeeding: 0, FeedingProbability: 0.5, LeaveTreeProbability: 0.25,
		},
	)
}
