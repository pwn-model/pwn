// Package config assembles an [app.App]'s resources and systems from a YAML
// config file, so that models can be composed without recompiling.
package config

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"sync"

	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
)

// registry maps a system's config type name to a factory that creates a
// zero-value instance of it, returned as an [app.System].
var registry = map[string]func() app.System{}

// Register makes a system type available for use in config files, under a
// name derived from the type itself: "<module>.<package>.<Type>", e.g.
// "pwn.sys.InitTrees" or "ark-tools.system.FixedTermination". There is
// nothing to name and nothing that can drift out of sync with the type.
//
// T is the system's concrete (value) type; PT must be *T and implement
// [app.System]. Call it as Register[sys.Colonization](), once per system
// type, typically from that type's own file's init function.
func Register[T any, PT interface {
	*T
	app.System
}]() {
	name := qualifiedName(reflect.TypeFor[T]())
	if _, ok := registry[name]; ok {
		panic(fmt.Sprintf("config: system type %q is already registered", name))
	}
	registry[name] = func() app.System {
		return PT(new(T))
	}
}

// newSystem creates a new, zero-value instance of the system type registered
// under name.
func newSystem(name string) (app.System, bool) {
	factory, ok := registry[name]
	if !ok {
		return nil, false
	}
	return factory(), true
}

// resourceEntry is a registered resource type: a way to create a decode
// target for its config shape, and a way to turn a decoded config value
// into the actual resource and add it to a world.
type resourceEntry struct {
	newConfig func() any
	apply     func(world *ecs.World, cfg any)
}

// resourceRegistry maps a resource's config type name to its resourceEntry.
var resourceRegistry = map[string]resourceEntry{}

// RegisterResource makes a resource type available for use in config files,
// under a name derived from the resource type T itself (see [Register]).
//
// Unlike systems, resources have no common construction shape (some need
// validation, derived fields, and so on), so registration takes a build
// function from a config shape C to the actual resource T; C is typically a
// plain, exported-field struct decoded directly from YAML, and build is
// often just the resource's own constructor. For a resource with no such
// distinction, use C == T and an identity build function.
//
// Call it as RegisterResource(func(c WorldSizeConfig) WorldSize { ... }),
// typically from the resource type's own file's init function. C and T are
// both inferred from build.
func RegisterResource[C any, T any](build func(C) T) {
	name := qualifiedName(reflect.TypeFor[T]())
	if _, ok := resourceRegistry[name]; ok {
		panic(fmt.Sprintf("config: resource type %q is already registered", name))
	}
	resourceRegistry[name] = resourceEntry{
		newConfig: func() any { return new(C) },
		apply: func(world *ecs.World, cfg any) {
			res := build(*cfg.(*C))
			ecs.AddResource(world, &res)
		},
	}
}

// newResourceConfig looks up the resource type registered under name.
func newResourceConfig(name string) (resourceEntry, bool) {
	entry, ok := resourceRegistry[name]
	return entry, ok
}

// moduleRoots lists all module paths involved in the build (this module and
// its dependencies), longest first, for longest-prefix matching in
// qualifiedName. Computed once, lazily, since it requires reading the
// binary's embedded build info.
var (
	moduleRootsOnce sync.Once
	moduleRoots     []string
)

func loadModuleRoots() {
	moduleRootsOnce.Do(func() {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return
		}
		moduleRoots = append(moduleRoots, info.Main.Path)
		for _, dep := range info.Deps {
			moduleRoots = append(moduleRoots, dep.Path)
		}
		sort.Slice(moduleRoots, func(i, j int) bool {
			return len(moduleRoots[i]) > len(moduleRoots[j])
		})
	})
}

// qualifiedName derives a config type-name for t of the form
// "<module>.<package...>.<Type>": the last path element of the Go module
// that declares t, followed by its package path relative to that module
// (with "/" replaced by "."), followed by its type name.
//
// E.g. for sys.InitTrees (module "github.com/pwn-model/pwn", package
// "sys"), this yields "pwn.sys.InitTrees".
func qualifiedName(t reflect.Type) string {
	loadModuleRoots()

	pkgPath := t.PkgPath()
	for _, root := range moduleRoots {
		if pkgPath == root {
			return lastPathElem(root) + "." + t.Name()
		}
		if rest, ok := strings.CutPrefix(pkgPath, root+"/"); ok {
			return lastPathElem(root) + "." + strings.ReplaceAll(rest, "/", ".") + "." + t.Name()
		}
	}

	// Fallback for when build info isn't available (e.g. some unusual build
	// modes): fully-qualified but still unique.
	return strings.ReplaceAll(pkgPath, "/", ".") + "." + t.Name()
}

func lastPathElem(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}
