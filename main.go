package main

import (
	"flag"
	"fmt"
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

	if err := RunFromConfig(*configPath); err != nil {
		log.Fatal(err)
	}
}

// RunFromConfig loads the model config file at configPath, builds an app
// from it, and runs it to completion. This is the entry point's own logic,
// factored out so it can be called from places other than main() (e.g.
// tests) without going through the command-line flag.
func RunFromConfig(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	a := app.New()
	a.TPS = cfg.TPS

	// The PRNG is hard-coded to Xoshiro256++, to stay bit-identical with
	// the sibling Julia implementation; only its seed is configurable.
	rnd := ecs.GetResource[resource.Rand](a.World)
	rnd.Source = res.NewXoshiro256pp(cfg.Seed)

	// Resources, systems and windows, all as configured.
	cfg.Apply(a)

	window.Run(a)

	//fmt.Println(util.TreesToString(a.World))

	return nil
}
