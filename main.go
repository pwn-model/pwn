package main

import (
	"flag"
	"log"

	"github.com/mlange-42/ark-pixel/window"
	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/config"
	_ "github.com/pwn-model/pwn/obs"
	_ "github.com/pwn-model/pwn/obs/maps"
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

	// Resources, systems and windows, all as configured.
	cfg.Apply(app)

	window.Run(app)

	//fmt.Println(util.TreesToString(app.World))
}
