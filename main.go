package main

import (
	"flag"
	"log"

	"github.com/mazznoer/colorgrad"
	"github.com/mlange-42/ark-pixel/plot"
	"github.com/mlange-42/ark-pixel/window"
	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/obs"
	"github.com/pwn-model/pwn/obs/maps"
	"github.com/pwn-model/pwn/res"
	_ "github.com/pwn-model/pwn/sys"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to the model config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	app := app.New()
	app.TPS = cfg.TPS

	// The PRNG is hard-coded to Xoshiro256++, to stay bit-identical with
	// the sibling Julia implementation; only its seed is configurable.
	rnd := ecs.GetResource[resource.Rand](app.World)
	rnd.Source = res.NewXoshiro256pp(cfg.Seed)

	// Resources and systems, all as configured.
	cfg.Apply(app)

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

	app.AddUISystem((&window.Window{}).
		With(&plot.Image{
			Observer: &maps.TreeColonization{CellSize: 100},
			Colors:   colorgrad.Viridis(),
			Max:      5,
		}))

	window.Run(app)

	//fmt.Println(util.TreesToString(app.World))
}
