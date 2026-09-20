package config

import (
	"fmt"
	"os"

	"github.com/mlange-42/ark-tools/app"
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
	// Resources lists the resources to add to the world. Each entry's
	// "type" selects the resource implementation (see [RegisterResource]);
	// its other fields are decoded into that resource's config shape.
	Resources []ResourceConfig `yaml:"resources"`
	// Systems lists the systems to add to the app, in order. Each entry's
	// "type" selects the system implementation (see [Register]); its other
	// fields are decoded directly into that system's exported parameters.
	Systems []SystemConfig `yaml:"systems"`
}

// entryType decodes the "type" field common to every systems/resources
// config entry, ahead of resolving and decoding the rest of it.
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
// then decodes the whole entry into the resulting concrete system, so that
// its own exported parameter fields are populated by the yaml package's
// usual struct decoding.
func (c *SystemConfig) UnmarshalYAML(node *yaml.Node) error {
	typ, err := entryType(node)
	if err != nil {
		return err
	}

	sys, ok := newSystem(typ)
	if !ok {
		return fmt.Errorf("unknown system type %q", typ)
	}
	if err := node.Decode(sys); err != nil {
		return fmt.Errorf("decoding system %q: %w", typ, err)
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

	entry, ok := newResourceConfig(typ)
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

// Apply adds all configured resources and systems to a, in order.
//
// It does not touch a's PRNG seed, nor UI systems/observers; those are
// still wired up by hand (see main.go).
func (c *Config) Apply(a *app.App) {
	for _, rc := range c.Resources {
		rc.apply(a.World)
	}
	for _, sc := range c.Systems {
		a.AddSystem(sc.System)
	}
}
