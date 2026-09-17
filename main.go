package main

import (
	"time"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/reporter"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark-tools/system"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/obs"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/sys"
)

func main() {
	app := app.New()
	app.TPS = 0

	// Resources
	rnd := ecs.GetResource[resource.Rand](app.World)
	rnd.Source = res.NewXoshiro256pp(uint64(time.Now().UnixNano()))

	worldSize := res.WorldSize{
		Width:      1000,
		Height:     1000,
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
	app.AddSystem(&sys.DiseaseCourse{
		TicksToDamage: 8,
	})
	app.AddSystem(&sys.RandomInfection{
		TickOfInfection: 0,
		NumTrees:        100,
		CellX:           10,
		CellY:           10,
	})

	// Observers
	app.AddSystem(&reporter.CSV{
		Observer: &obs.TreePopulationObserver{},
		File:     "out/tree_pop.csv",
	})

	// Stop criterion
	app.AddSystem(&system.FixedTermination{Steps: 100})

	app.Run()
}
