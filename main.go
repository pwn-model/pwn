package main

import (
	"time"

	"github.com/mlange-42/ark-pixel/plot"
	"github.com/mlange-42/ark-pixel/window"
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
	app.TPS = 30

	// Resources
	rnd := ecs.GetResource[resource.Rand](app.World)
	rnd.Source = res.NewXoshiro256pp(uint64(time.Now().UnixNano()))

	worldSize := res.WorldSize{
		Width:      200,
		Height:     200,
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
		CellX:           2,
		CellY:           2,
	})

	// Observers
	app.AddSystem(&reporter.CSV{
		Observer: &obs.TreePopulation{},
		File:     "out/tree_pop.csv",
	})

	app.AddUISystem((&window.Window{}).
		With(&plot.TimeSeries{
			Observer: &obs.TreeDamage{},
		}))

	// Stop criterion
	app.AddSystem(&system.FixedTermination{Steps: 1000})

	window.Run(app)
}
