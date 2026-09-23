package main

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark-tools/system"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/sys"
)

func setupAndRun(b *testing.B) {
	for b.Loop() {
		app := setup(10000, 10000, 520)
		app.Run()
	}
}

func setupOnly(b *testing.B) {
	for b.Loop() {
		_ = setup(10000, 10000, 520)
	}
}

func runOnly(b *testing.B) {
	for b.Loop() {
		b.StopTimer()
		app := setup(10000, 10000, 520)
		b.StartTimer()
		app.Run()
	}
}

func setup(width, height, ticks int) *app.App {
	app := app.New()
	app.TPS = 0

	// Resources
	rnd := ecs.GetResource[resource.Rand](app.World)
	rnd.Source = res.NewXoshiro256pp(1)

	worldSize := res.NewWorldSize(width, height, 10, 500)
	ecs.AddResource(app.World, &worldSize)

	// Initialization
	app.AddSystem(&sys.InitGrids{})
	app.AddSystem(&sys.InitTrees{
		CellProbability:  1.0,
		TreeProbability:  0.9,
		DamagePrevalence: 0.03,
		BeetlePrevalence: 0.2,
	})

	// Systems
	app.AddSystem(&sys.UpdateTime{
		TicksPerYear: 52,
	})
	app.AddSystem(&sys.DiseaseCourse{
		TicksToDamage: 8,
	})
	app.AddSystem(&sys.RandomInfection{
		TickOfInfection: 0,
		NumTrees:        100,
		CellX:           10,
		CellY:           10,
	})
	app.AddSystem(&sys.DamageTrees{
		TickOfYear:         35,
		DamageProbability:  0.01,
		RemovalProbability: 0.333,
	})
	app.AddSystem(&sys.BeetleEmergence{
		TickOfYear:     19,
		BeetlesPerTree: 10,
		LifeExpectancy: 5,
	})
	app.AddSystem(&sys.TreeAttraction{
		TickOfYear:    18,
		DamagedTrees:  true,
		HalfDistance:  50,
		DensityRadius: 20,
		DensityWeight: 1.0,
	})
	app.AddSystem(&sys.TreeAttraction{
		TickOfYear:    18,
		DamagedTrees:  false,
		HalfDistance:  50,
		DensityRadius: 20,
		DensityWeight: 1.0,
	})
	app.AddSystem(&sys.BeetleMovement{
		StepsPerTick:          7,
		DurationFeeding:       12,
		DurationEggLaying:     6,
		LeaveTreeProbability:  0.5,
		RandomWalkProbability: 0.5,
	})
	app.AddSystem(&sys.BeetleMortality{})
	app.AddSystem(&sys.Colonization{
		TickOfYear:         20,
		CellSize:           100,
		KernelHalfDistance: 50,
		KernelRadius:       300,
		BeetlesPerTree:     2.2,
		TreesPerBeetle:     1,
	})

	// Stop criterion
	app.AddSystem(&system.FixedTermination{Steps: int64(ticks)})

	app.Initialize()

	return app
}
