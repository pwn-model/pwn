package config_test

import (
	"testing"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/resource"
	"github.com/mlange-42/ark-tools/system"
	"github.com/mlange-42/ark/ecs"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/obs"
	"github.com/pwn-model/pwn/obs/maps"
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

// TestConfig_Apply builds and runs an app to completion, as an end-to-end
// smoke test of resource setup, system ordering and the stop criterion.
//
// Deliberately built from an inline config, not the repo's own
// scripts/config.yaml: that one also declares "windows", and Apply adds
// those as real UI systems whose Initialize opens an actual OpenGL window,
// which a headless test must not trigger. See main_test.go for a
// windows-aware check of the real file, without running it.
func TestConfig_Apply(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
seed: 1
resources:
  - type: pwn.res.WorldSize
    width: 4000
    height: 3000
    cell_size: 10
    grid_cell_size: 500
systems:
  - type: pwn.sys.InitGrids
  - type: ark-tools.system.FixedTermination
    steps: 10
`), &cfg)
	require.NoError(t, err)

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
	require.True(t, ok)

	a.Run()

	tick := ecs.GetResource[resource.Tick](a.World)
	assert.Equal(t, term.Steps, tick.Tick)
}

func TestConfig_UnmarshalWindows(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
windows:
  - title: Tree colonization
    draw_interval: 2
    drawers:
      - type: pwn.obs.maps.Trees
`), &cfg)
	require.NoError(t, err)
	require.Len(t, cfg.Windows, 1)

	win := cfg.Windows[0]
	assert.Equal(t, "Tree colonization", win.Title)
	assert.Equal(t, 2, win.DrawInterval)
	require.Len(t, win.Drawers, 1)

	_, ok := win.Drawers[0].Drawer.(*maps.Trees)
	assert.True(t, ok)
}

func TestConfig_UnmarshalWindows_UnknownDrawerType(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
windows:
  - drawers:
      - type: NoSuchDrawer
`), &cfg)
	assert.ErrorContains(t, err, "NoSuchDrawer")
}

func TestConfig_UnmarshalWindows_MissingDrawerType(t *testing.T) {
	var cfg config.Config
	err := yaml.Unmarshal([]byte(`
windows:
  - drawers:
      - scale: 1.0
`), &cfg)
	assert.ErrorContains(t, err, "type")
}

// TestConfig_NestedObservers resolves a Row and a Matrix observer nested
// inside RowObserverConfig/MatrixObserverConfig fields directly, the same
// way a drawer or reporter's own "observer" field would.
func TestConfig_NestedObservers(t *testing.T) {
	var row config.RowObserverConfig
	err := yaml.Unmarshal([]byte(`type: pwn.obs.TreeColonization`), &row)
	require.NoError(t, err)
	_, ok := row.Row.(*obs.TreeColonization)
	assert.True(t, ok)

	var matrix config.MatrixObserverConfig
	err = yaml.Unmarshal([]byte(`
type: pwn.obs.maps.TreeColonization
cell_size: 100
`), &matrix)
	require.NoError(t, err)
	m, ok := matrix.Matrix.(*maps.TreeColonization)
	require.True(t, ok)
	assert.Equal(t, 100, m.CellSize)
}

// TestConfig_NestedObservers_WrongKind proves that sharing one registry
// between Row and Matrix observers (see RegisterObserver) doesn't weaken
// checking: pwn.obs.TreeColonization only implements observer.Row, so
// using it where a Matrix observer is expected must still fail, just at
// resolution time (via RegisterObserver/RowObserverConfig's own type
// assertion) instead of at registration time.
func TestConfig_NestedObservers_WrongKind(t *testing.T) {
	var matrix config.MatrixObserverConfig
	err := yaml.Unmarshal([]byte(`type: pwn.obs.TreeColonization`), &matrix)
	assert.ErrorContains(t, err, "pwn.obs.TreeColonization")
}

func TestConfig_NestedObservers_UnknownType(t *testing.T) {
	var row config.RowObserverConfig
	err := yaml.Unmarshal([]byte(`type: NoSuchObserver`), &row)
	assert.ErrorContains(t, err, "NoSuchObserver")

	var matrix config.MatrixObserverConfig
	err = yaml.Unmarshal([]byte(`type: NoSuchObserver`), &matrix)
	assert.ErrorContains(t, err, "NoSuchObserver")
}
