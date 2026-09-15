package main

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/system"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/sys"
)

func BenchmarkAll(b *testing.B) {
	for b.Loop() {
		app := setup(100, 100, 100)
		app.Run()
	}
}

func BenchmarkSetup(b *testing.B) {
	for b.Loop() {
		_ = setup(100, 100, 100)
	}
}

// func BenchmarkRun(b *testing.B) {
// 	for b.Loop() {
// 		b.StopTimer()
// 		app := setup(100, 100, 100)
// 		b.StartTimer()
// 		app.Run()
// 	}
// }

func setup(width, height, ticks int) *app.App {
	app := app.New()
	app.TPS = 0

	treeGrid := res.NewEntityGrid(width, height)
	ecs.AddResource(app.World, &treeGrid)

	app.AddSystem(&sys.InitTrees{TreeProbability: 0.9})
	app.AddSystem(&system.FixedTermination{Steps: int64(ticks)})

	app.Initialize()

	return app
}
