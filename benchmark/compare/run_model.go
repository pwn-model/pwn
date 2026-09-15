package main

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
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

// func runOnly(b *testing.B) {
// 	for b.Loop() {
// 		b.StopTimer()
// 		app := setup(1000, 1000, 100)
// 		b.StartTimer()
// 		app.Run()
// 	}
// }

func setup(width, height, ticks int) *app.App {
	app := app.New()
	app.TPS = 0

	worldSize := res.WorldSize{
		Width:      width,
		Height:     height,
		Resolution: 50,
	}
	ecs.AddResource(app.World, &worldSize)

	app.AddSystem(&sys.InitGrids{})
	app.AddSystem(&sys.InitTrees{TreeProbability: 0.9})
	app.AddSystem(&system.FixedTermination{Steps: int64(ticks)})

	app.Initialize()

	return app
}
