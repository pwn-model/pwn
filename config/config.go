package config

import (
	"fmt"
	"os"

	"github.com/mlange-42/ark-pixel/window"
	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark-tools/observer"
	"github.com/mlange-42/ark/ecs"
	yaml "go.yaml.in/yaml/v3"
)

// Config is the top-level, deserialized shape of a model config file.
type Config struct {
	// Seed for the model's PRNG. The PRNG implementation itself is fixed
	// (not configurable), to keep it bit-identical with the sibling Julia
	// implementation.
	Seed uint64 `yaml:"seed"`
	// TPS is the app's ticks-per-second cap for normal systems (see
	// [app.Systems].TPS). Values <= 0 mean as fast as possible.
	TPS float64 `yaml:"tps"`
	// FPS is the app's frames-per-second cap for UI systems.
	// Values <= 0 mean as fast as possible.
	FPS float64 `yaml:"fps"`
	// Resources lists the resources to add to the world. Each entry's
	// "type" selects the resource implementation (see [RegisterResource]);
	// its other fields are decoded into that resource's config shape.
	Resources []ResourceConfig `yaml:"resources"`
	// Systems lists the systems to add to the app, in order. Each entry's
	// "type" selects the system implementation (see [Register]); its other
	// fields are decoded directly into that system's exported parameters.
	Systems []SystemConfig `yaml:"systems"`
	// Windows lists the UI windows to open, each with its own list of
	// drawers (see [WindowConfig]).
	Windows []WindowConfig `yaml:"windows"`
}

// entryType decodes the "type" field common to every systems/resources/
// drawers/observers config entry, ahead of resolving and decoding the rest
// of it.
func entryType(node *yaml.Node) (string, error) {
	var head struct {
		Type string `yaml:"type"`
	}
	if err := node.Decode(&head); err != nil {
		return "", err
	}
	if head.Type == "" {
		return "", fmt.Errorf("config entry is missing a 'type' field")
	}
	return head.Type, nil
}

// SystemConfig decodes a single "type" + parameters entry from the
// "systems" list into a concrete, registered system.
type SystemConfig struct {
	System app.System
}

// UnmarshalYAML resolves the entry's "type" field via the system registry,
// then decodes the rest of the entry into that system's config shape.
func (c *SystemConfig) UnmarshalYAML(node *yaml.Node) error {
	sys, err := buildFromNode[app.System](node, registry)
	if err != nil {
		return err
	}
	c.System = sys
	return nil
}

// ResourceConfig decodes a single "type" + parameters entry from the
// "resources" list, deferring actually building and adding the resource
// until [Config.Apply] has a world to add it to.
type ResourceConfig struct {
	apply func(world *ecs.World)
}

// UnmarshalYAML resolves the entry's "type" field via the resource
// registry, then decodes the whole entry into that resource type's config
// shape.
func (c *ResourceConfig) UnmarshalYAML(node *yaml.Node) error {
	typ, err := entryType(node)
	if err != nil {
		return err
	}

	entry, ok := resourceRegistry[typ]
	if !ok {
		return fmt.Errorf("unknown resource type %q", typ)
	}

	cfg := entry.newConfig()
	if err := node.Decode(cfg); err != nil {
		return fmt.Errorf("decoding resource %q: %w", typ, err)
	}

	c.apply = func(world *ecs.World) { entry.apply(world, cfg) }
	return nil
}

// DrawerConfig decodes a single "type" + parameters entry from a window's
// "drawers" list into a concrete, registered [window.Drawer].
type DrawerConfig struct {
	Drawer window.Drawer
}

// UnmarshalYAML resolves the entry's "type" field via the drawer registry,
// then decodes the rest of the entry into that drawer's config shape.
func (c *DrawerConfig) UnmarshalYAML(node *yaml.Node) error {
	d, err := buildFromNode[window.Drawer](node, drawerRegistry)
	if err != nil {
		return err
	}
	c.Drawer = d
	return nil
}

// RowObserverConfig decodes a single "type" + parameters entry, used as a
// nested "observer" field of a drawer or reporter, into a concrete,
// registered [observer.Row].
type RowObserverConfig struct {
	Row observer.Row
}

// UnmarshalYAML resolves the entry's "type" field via the observer
// registry, then decodes the rest of the entry into that observer's config
// shape. Resolution fails if the named type doesn't implement [observer.Row]
// (see [RegisterObserver]).
func (c *RowObserverConfig) UnmarshalYAML(node *yaml.Node) error {
	o, err := buildFromNode[observer.Row](node, observerRegistry)
	if err != nil {
		return err
	}
	c.Row = o
	return nil
}

// MatrixObserverConfig decodes a single "type" + parameters entry, used as
// a nested "observer" field of a drawer, into a concrete, registered
// [observer.Matrix].
type MatrixObserverConfig struct {
	Matrix observer.Matrix
}

// UnmarshalYAML resolves the entry's "type" field via the observer
// registry, then decodes the rest of the entry into that observer's config
// shape. Resolution fails if the named type doesn't implement
// [observer.Matrix] (see [RegisterObserver]).
func (c *MatrixObserverConfig) UnmarshalYAML(node *yaml.Node) error {
	o, err := buildFromNode[observer.Matrix](node, observerRegistry)
	if err != nil {
		return err
	}
	c.Matrix = o
	return nil
}

// WindowConfig is a single entry of the top-level "windows" list: a UI
// window and its ordered list of drawers. Unlike systems/resources/
// drawers/observers, there is only ever one window implementation
// ([window.Window]), so a WindowConfig needs no "type" field or registry of
// its own; only its (polymorphic) Drawers need one.
type WindowConfig struct {
	Title        string         `yaml:"title"`
	Bounds       window.Bounds  `yaml:"bounds"`
	DrawInterval int            `yaml:"draw_interval"`
	Drawers      []DrawerConfig `yaml:"drawers"`
}

// build turns wc into an actual, ready-to-add *window.Window.
func (wc WindowConfig) build() *window.Window {
	w := &window.Window{
		Title:        wc.Title,
		Bounds:       wc.Bounds,
		DrawInterval: wc.DrawInterval,
	}
	for _, dc := range wc.Drawers {
		w.With(dc.Drawer)
	}
	return w
}

// Load reads and parses a model config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %q: %w", path, err)
	}
	return &cfg, nil
}

// Apply adds all configured resources, systems and UI windows to a, in
// order.
//
// It does not touch a's PRNG seed; that is still wired up by hand (see
// main.go), to keep the PRNG implementation itself fixed.
func (c *Config) Apply(a *app.App) {
	for _, rc := range c.Resources {
		rc.apply(a.World)
	}
	for _, sc := range c.Systems {
		a.AddSystem(sc.System)
	}
	for _, wc := range c.Windows {
		a.AddUISystem(wc.build())
	}
}
