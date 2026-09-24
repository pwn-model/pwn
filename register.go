package main

import (
	"fmt"
	"time"

	"github.com/mazznoer/colorgrad"
	"github.com/mlange-42/ark-pixel/monitor"
	"github.com/mlange-42/ark-pixel/plot"
	"github.com/mlange-42/ark-tools/reporter"
	"github.com/pwn-model/pwn/config"
)

// Registers the drawer/reporter types that wrap external plotting/reporting
// libraries (ark-pixel/plot, ark-tools/reporter) for use from config files.
// These are only relevant to this interactive binary (not to headless
// benchmarks or tests), which is why they live here rather than in sys/ or
// obs/ alongside the plain, project-owned systems and observers that
// self-register in their own files.

// colorGradients are the color-gradient presets selectable by name for a
// plot.Image's "colors" field. Extend by adding another colorgrad preset.
var colorGradients = map[string]colorgrad.Gradient{
	"viridis": colorgrad.Viridis(),
	"turbo":   colorgrad.Turbo(),
	"plasma":  colorgrad.Plasma(),
	"inferno": colorgrad.Inferno(),
	"magma":   colorgrad.Magma(),
	"cividis": colorgrad.Cividis(),
}

func resolveGradient(name string) colorgrad.Gradient {
	if name == "" {
		return colorgrad.Viridis()
	}
	g, ok := colorGradients[name]
	if !ok {
		panic(fmt.Sprintf("unknown color gradient %q", name))
	}
	return g
}

// timeSeriesConfig is plot.TimeSeries's config shape: identical, except its
// Observer is resolved via the Row observer registry instead of being a
// bare, undecodable observer.Row field.
type timeSeriesConfig struct {
	Observer       config.RowObserverConfig `yaml:"observer"`
	Columns        []string                 `yaml:"columns"`
	UpdateInterval int                      `yaml:"update_interval"`
	Labels         plot.Labels              `yaml:"labels"`
	MaxRows        int                      `yaml:"max_rows"`
}

// imageConfig is plot.Image's config shape: its Observer is resolved via
// the Matrix observer registry, and Colors is a color gradient preset name
// instead of a bare colorgrad.Gradient.
type imageConfig struct {
	Scale    float64                     `yaml:"scale"`
	Observer config.MatrixObserverConfig `yaml:"observer"`
	Colors   string                      `yaml:"colors"`
	Min      float64                     `yaml:"min"`
	Max      float64                     `yaml:"max"`
}

// monitorConfig is monitor.Monitor's config shape: identical, but with
// snake_case field names. SampleInterval is a duration string like "500ms".
type monitorConfig struct {
	PlotCapacity   int           `yaml:"plot_capacity"`
	SampleInterval time.Duration `yaml:"sample_interval"`
	HidePlots      bool          `yaml:"hide_plots"`
	HideArchetypes bool          `yaml:"hide_archetypes"`
}

// csvReporterConfig is reporter.CSV's config shape: its Observer is
// resolved via the Row observer registry.
type csvReporterConfig struct {
	Observer       config.RowObserverConfig `yaml:"observer"`
	File           string                   `yaml:"file"`
	Sep            string                   `yaml:"sep"`
	UpdateInterval int                      `yaml:"update_interval"`
	Final          bool                     `yaml:"final"`
}

func init() {
	config.RegisterDrawerFunc(func(c timeSeriesConfig) *plot.TimeSeries {
		return &plot.TimeSeries{
			Observer:       c.Observer.Row,
			Columns:        c.Columns,
			UpdateInterval: c.UpdateInterval,
			Labels:         c.Labels,
			MaxRows:        c.MaxRows,
		}
	})

	config.RegisterDrawerFunc(func(c imageConfig) *plot.Image {
		return &plot.Image{
			Scale:    c.Scale,
			Observer: c.Observer.Matrix,
			Colors:   resolveGradient(c.Colors),
			Min:      c.Min,
			Max:      c.Max,
		}
	})

	config.RegisterDrawerFunc(func(c monitorConfig) *monitor.Monitor {
		return &monitor.Monitor{
			PlotCapacity:   c.PlotCapacity,
			SampleInterval: c.SampleInterval,
			HidePlots:      c.HidePlots,
			HideArchetypes: c.HideArchetypes,
		}
	})

	config.RegisterDrawer[monitor.Controls]()

	config.RegisterFunc(func(c csvReporterConfig) *reporter.CSV {
		return &reporter.CSV{
			Observer:       c.Observer.Row,
			File:           c.File,
			Sep:            c.Sep,
			UpdateInterval: c.UpdateInterval,
			Final:          c.Final,
		}
	})
}
