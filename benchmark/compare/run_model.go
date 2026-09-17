package main

import (
	"testing"
	"time"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark-tools/system"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/sys"
)

func setupAndRun(b *testing.B) {
	for b.Loop() {
		app := setup(1000, 1000, 100)
		app.Run()
	}
}

func setupOnly(b *testing.B) {
	for b.Loop() {
		_ = setup(1000, 1000, 100)
	}
}

func runOnly(b *testing.B) {
	for b.Loop() {
		b.StopTimer()
		app := setup(1000, 1000, 100)
		b.StartTimer()
		app.Run()
	}
}

func setup(width, height, ticks int) *app.App {
	app := app.New()
	app.TPS = 0

	// Resources
	rnd := ecs.GetResource[resource.Rand](app.World)
	rnd.Source = res.NewXoshiro256pp(uint64(time.Now().UnixNano()))

	worldSize := res.WorldSize{
		Width:      width,
		Height:     height,
		Resolution: 50,
	}
	ecs.AddResource(app.World, &worldSize)

	// Initialization
	app.AddSystem(&sys.InitGrids{})
	app.AddSystem(&sys.InitTrees{
		TreeProbability:  0.9,
		DamagePrevalence: 0.03,
	})

	// Systems
	app.AddSystem(&sys.UpdateTime{
		WeeksPerYear: 52,
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

	// Stop criterion
	app.AddSystem(&system.FixedTermination{Steps: int64(ticks)})

	app.Initialize()

	return app
}
