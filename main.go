package main

import (
	"github.com/mlange-42/ark-pixel/plot"
	"github.com/mlange-42/ark-pixel/window"
	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark-tools/system"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/obs"
	"github.com/pwn-model/pwn/obs/maps"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/sys"
)

func main() {
	app := app.New()
	app.TPS = 30

	// Resources
	rnd := ecs.GetResource[resource.Rand](app.World)
	rnd.Source = res.NewXoshiro256pp(1)

	worldSize := res.WorldSize{
		Width:      400,
		Height:     300,
		Resolution: 50,
	}
	ecs.AddResource(app.World, &worldSize)

	// Initialization
	app.AddSystem(&sys.InitGrids{})
	app.AddSystem(&sys.InitTrees{
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
		CellX:           2,
		CellY:           2,
	})
	app.AddSystem(&sys.DamageTrees{
		TickOfYear:         35,
		DamageProbability:  0.01,
		RemovalProbability: 0.333,
	})
	app.AddSystem(&sys.Colonization{
		TickOfYear:     20,
		KernelScale:    1.0,
		KernelRadius:   5,
		BeetlesPerTree: 2.2,
		TreesPerBeetle: 1.0,
	})

	// Observers

	app.AddUISystem((&window.Window{}).
		With(&plot.TimeSeries{
			Observer: &obs.TreeColonization{},
			Labels: plot.Labels{
				Title: "Tree colonization",
				X:     "time [weeks]",
				Y:     "proportion colonized",
			},
		}))

	app.AddUISystem((&window.Window{}).
		With(&maps.Trees{}))

	// Stop criterion
	app.AddSystem(&system.FixedTermination{Steps: 5200})

	window.Run(app)

	//fmt.Println(util.TreesToString(app.World))
}
