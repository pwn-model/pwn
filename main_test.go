package main

import (
	"testing"

	"github.com/mlange-42/ark-pixel/plot"
	"github.com/mlange-42/ark-tools/reporter"
	"github.com/pwn-model/pwn/config"
	"github.com/pwn-model/pwn/obs"
	"github.com/pwn-model/pwn/obs/maps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadShippedConfig loads the repo's own config.yaml and checks its
// overall shape, including the "windows" and reporter sections, without
// calling Config.Apply/app.Run: a window's Initialize opens a real OpenGL
// window, which this headless test suite must not do. The registrations
// this exercises (plot.TimeSeries, plot.Image, reporter.CSV) only happen
// in this package (see register_visuals.go), which is why this test lives
// here rather than in the config package's own tests.
func TestLoadShippedConfig(t *testing.T) {
	cfg, err := config.Load("config.yaml")
	require.NoError(t, err)

	assert.Equal(t, uint64(1), cfg.Seed)
	assert.Equal(t, 30.0, cfg.TPS)

	var csv *reporter.CSV
	for _, sc := range cfg.Systems {
		if c, ok := sc.System.(*reporter.CSV); ok {
			csv = c
		}
	}
	require.NotNil(t, csv, "config.yaml is expected to have a CSV reporter system")
	assert.Equal(t, "out/tree_pop.csv", csv.File)
	_, ok := csv.Observer.(*obs.TreePopulation)
	assert.True(t, ok)

	require.Len(t, cfg.Windows, 6)

	timeSeriesWindow := cfg.Windows[0]
	require.Len(t, timeSeriesWindow.Drawers, 1)
	ts, ok := timeSeriesWindow.Drawers[0].Drawer.(*plot.TimeSeries)
	require.True(t, ok)
	_, ok = ts.Observer.(*obs.TreeColonization)
	assert.True(t, ok)
	assert.Equal(t, "Tree colonization", ts.Labels.Title)

	treesWindow := cfg.Windows[1]
	require.Len(t, treesWindow.Drawers, 1)
	_, ok = treesWindow.Drawers[0].Drawer.(*maps.Trees)
	assert.True(t, ok)

	imageWindow := cfg.Windows[2]
	require.Len(t, imageWindow.Drawers, 1)
	img, ok := imageWindow.Drawers[0].Drawer.(*plot.Image)
	require.True(t, ok)
	colonizationMap, ok := img.Observer.(*maps.TreeColonization)
	require.True(t, ok)
	assert.Equal(t, 100, colonizationMap.CellSize)
	assert.Equal(t, 5.0, img.Max)
}
