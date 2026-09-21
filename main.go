package main

import (
	"fmt"
	"log"

	"github.com/alecthomas/kong"
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

// CLI is the command-line interface of the model executable.
type CLI struct {
	Config string `arg:"" optional:"" default:"config.yaml" type:"path" help:"Path to the model config file. Default: config.yaml"`
}

func main() {
	var cli CLI
	kong.Parse(&cli, kong.Description("Runs the Pine Wilt Nematode model."))

	if err := RunFromConfig(cli.Config); err != nil {
		log.Fatal(err)
	}
}

// RunFromConfig loads the model config file at configPath, builds an app
// from it, and runs it to completion. This is the entry point's own logic,
// factored out so it can be called from places other than main() (e.g.
// tests) without going through the command line.
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
