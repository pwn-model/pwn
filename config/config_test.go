package config_test

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark-tools/system"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/res"
	"github.com/pwn-model/pwn/sys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v3"
)

func TestConfig_UnmarshalSystems(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
systems:
  - type: pwn.sys.InitGrids
  - type: pwn.sys.InitTrees
    tree_probability: 0.9
    damage_prevalence: 0.03
    beetle_prevalence: 0.2
`), &cfg)
	require.NoError(t, err)
	require.Len(t, cfg.Systems, 2)

	_, ok := cfg.Systems[0].System.(*sys.InitGrids)
	assert.True(t, ok)

	initTrees, ok := cfg.Systems[1].System.(*sys.InitTrees)
	require.True(t, ok)
	assert.Equal(t, 0.9, initTrees.TreeProbability)
	assert.Equal(t, 0.03, initTrees.DamagePrevalence)
	assert.Equal(t, 0.2, initTrees.BeetlePrevalence)
}

func TestConfig_UnmarshalSystems_UnknownType(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
systems:
  - type: NoSuchSystem
`), &cfg)
	assert.ErrorContains(t, err, "NoSuchSystem")
}

func TestConfig_UnmarshalSystems_MissingType(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
systems:
  - steps: 10
`), &cfg)
	assert.ErrorContains(t, err, "type")
}

// TestConfig_Resources unmarshals and applies a "resources" entry, since a
// resource's decoded config is otherwise opaque until applied to a world
// (unlike a system, it isn't exposed as a concrete, inspectable type).
func TestConfig_Resources(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
resources:
  - type: pwn.res.WorldSize
    width: 4000
    height: 3000
    cell_size: 10
    grid_cell_size: 500
`), &cfg)
	require.NoError(t, err)
	require.Len(t, cfg.Resources, 1)

	a := app.New()
	cfg.Apply(a)

	ws := ecs.GetResource[res.WorldSize](a.World)
	assert.Equal(t, 400, ws.Width())
	assert.Equal(t, 300, ws.Height())
	assert.Equal(t, 10, ws.CellSize())
	assert.Equal(t, 50, ws.Resolution())
}

func TestConfig_UnmarshalResources_UnknownType(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
resources:
  - type: NoSuchResource
`), &cfg)
	assert.ErrorContains(t, err, "NoSuchResource")
}

func TestConfig_UnmarshalResources_MissingType(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
resources:
  - width: 10
`), &cfg)
	assert.ErrorContains(t, err, "type")
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := config.Load("does_not_exist.yaml")
	assert.Error(t, err)
}

// TestConfig_Apply builds an app from the repo's own config.yaml, the same
// way main.go does, and runs it to completion, as an end-to-end smoke test
// of resource setup, system ordering and the stop criterion. The PRNG seed
// itself is applied by main.go, not by Config.Apply (see main.go), so it's
// set here the same way.
func TestConfig_Apply(t *testing.T) {
	cfg, err := config.Load("../config.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, cfg.Systems)
	assert.Equal(t, uint64(1), cfg.Seed)
	assert.Equal(t, 30.0, cfg.TPS)

	// Note: intentionally not setting a.TPS = cfg.TPS here (unlike main.go).
	// TPS throttles Run() to real time, which would make this test take
	// minutes; correctness doesn't depend on pacing.
	a := app.New()

	rnd := ecs.GetResource[resource.Rand](a.World)
	rnd.Source = res.NewXoshiro256pp(cfg.Seed)

	cfg.Apply(a)

	ws := ecs.GetResource[res.WorldSize](a.World)
	assert.Equal(t, 400, ws.Width())

	last := cfg.Systems[len(cfg.Systems)-1].System
	term, ok := last.(*system.FixedTermination)
	require.True(t, ok, "config.yaml is expected to end with a FixedTermination entry")

	a.Run()

	tick := ecs.GetResource[resource.Tick](a.World)
	assert.Equal(t, term.Steps, tick.Tick)
}
