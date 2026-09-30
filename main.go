package main

import (
	"fmt"
	"log"
	"os"

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
	OutDir string `short:"o" type:"path" help:"Directory to run the model in. Default: ."`
}

func main() {
	var cli CLI
	kong.Parse(&cli, kong.Description("Runs the Pine Wilt Nematode model."))

	if err := RunFromConfig(cli.Config, cli.OutDir); err != nil {
		log.Fatal(err)
	}
}

// RunFromConfig loads the model config file at configPath, builds an app
// from it, and runs it to completion. This is the entry point's own logic,
// factored out so it can be called from places other than main() (e.g.
// tests) without going through the command line.
//
// If outDir is not empty, the model runs with outDir (created if missing)
// as the working directory, so that all relative paths in the config (e.g.
// output files) are resolved against it. configPath itself is resolved
// against the original working directory, which is restored afterwards.
func RunFromConfig(configPath string, outDir string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if outDir != "" {
		restore, err := chdir(outDir)
		if err != nil {
			return err
		}
		defer restore()
	}

	a := app.New()
	a.TPS = cfg.TPS
	a.FPS = cfg.FPS

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

// chdir creates dir if missing and makes it the working directory. It
// returns a function that restores the previous working directory.
func chdir(dir string) (func(), error) {
	prev, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting working directory: %w", err)
	}
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}
	if err := os.Chdir(dir); err != nil {
		return nil, fmt.Errorf("changing to output directory: %w", err)
	}
	return func() {
		if err := os.Chdir(prev); err != nil {
			log.Printf("restoring working directory: %v", err)
		}
	}, nil
}
