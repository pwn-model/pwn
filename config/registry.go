// Package config assembles an [app.App]'s resources, systems and UI windows
// from a YAML config file, so that models can be composed without
// recompiling.
package config

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"sort"
	"strings"
	"sync"

	"github.com/mlange-42/ark-pixel/window"
	"github.com/mlange-42/ark-tools/app"
	"github.com/mlange-42/ark/ecs"
	yaml "go.yaml.in/yaml/v3"
)

// buildEntry is a registered type's way of turning a decoded config value
// into the actual, usable value: a fresh decode target for its config
// shape, and a way to build the real thing from it.
//
// It is the shared shape behind every registry in this file except
// resources (see resourceEntry): systems, drawers and observers are all
// just handed back as a value implementing some interface, with no extra
// world-mutating step the way adding a resource needs.
type buildEntry struct {
	newConfig func() any        // fresh *C for yaml to decode into
	build     func(cfg any) any // *C -> the constructed value
}

// register records, in reg under T's qualifiedName, a way to decode a
// config shape C and turn it into a T via build.
func register[C any, T any](reg map[string]buildEntry, build func(C) T) {
	name := qualifiedName(reflect.TypeFor[T]())
	if _, ok := reg[name]; ok {
		panic(fmt.Sprintf("config: type %q is already registered", name))
	}
	reg[name] = buildEntry{
		newConfig: func() any { return new(C) },
		build:     func(cfg any) any { return build(*cfg.(*C)) },
	}
}

// buildFromNode resolves a "type" + parameters YAML node via reg, decodes
// the rest of the node into the resolved type's config shape, and builds
// the final value, asserting it implements I.
func buildFromNode[I any](node *yaml.Node, reg map[string]buildEntry) (I, error) {
	var zero I

	typ, err := entryType(node)
	if err != nil {
		return zero, err
	}

	entry, ok := reg[typ]
	if !ok {
		return zero, fmt.Errorf("unknown type %q", typ)
	}

	cfg := entry.newConfig()
	if err := node.Decode(cfg); err != nil {
		return zero, fmt.Errorf("decoding %q: %w", typ, err)
	}

	v, ok := entry.build(cfg).(I)
	if !ok {
		return zero, fmt.Errorf("%q does not implement the expected interface", typ)
	}
	return v, nil
}

// registry maps a system's config type name to its buildEntry.
var registry = map[string]buildEntry{}

// Register makes a system type available for use in config files, under a
// name derived from the type itself: "<module>.<package>.<Type>", e.g.
// "pwn.sys.InitTrees" or "ark-tools.system.FixedTermination". There is
// nothing to name and nothing that can drift out of sync with the type.
//
// T is the system's concrete (value) type; PT must be *T and implement
// [app.System]. Call it as Register[sys.Colonization](), once per system
// type, typically from that type's own file's init function.
//
// For a system that needs more than its own exported fields decoded
// directly (e.g. one with a nested, polymorphic field), use [RegisterFunc]
// instead.
func Register[T any, PT interface {
	*T
	app.System
}]() {
	register(registry, func(c T) PT { return PT(&c) })
}

// RegisterFunc makes a system type available for use in config files (see
// [Register]), by decoding a config shape C and turning it into the system
// via build. Use this when T's own fields aren't enough to decode directly,
// e.g. because one of them is itself a nested, polymorphic value such as a
// [window.Drawer] or [observer.Row]/[observer.Matrix] (see
// [RowObserverConfig], [MatrixObserverConfig]).
func RegisterFunc[C any, T app.System](build func(C) T) {
	register(registry, build)
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

// drawerRegistry maps a drawer's config type name to its buildEntry.
var drawerRegistry = map[string]buildEntry{}

// RegisterDrawer makes a [window.Drawer] type available for use in a
// window's "drawers" list (see [Register]).
//
// For a drawer that needs more than its own exported fields decoded
// directly, use [RegisterDrawerFunc] instead.
func RegisterDrawer[T any, PT interface {
	*T
	window.Drawer
}]() {
	register(drawerRegistry, func(c T) PT { return PT(&c) })
}

// RegisterDrawerFunc makes a [window.Drawer] type available for use in a
// window's "drawers" list, by decoding a config shape C and turning it into
// the drawer via build (see [RegisterFunc]).
func RegisterDrawerFunc[C any, T window.Drawer](build func(C) T) {
	register(drawerRegistry, build)
}

// observerRegistry maps an observer's config type name to its buildEntry.
//
// There is one registry for both observer kinds
// ([observer.Row]/[observer.Matrix] from github.com/mlange-42/ark-tools/observer),
// not one per kind: a registered type isn't "a Row observer" or "a Matrix
// observer" in the abstract, it's just whatever it implements, and that's
// exactly what [RowObserverConfig]/[MatrixObserverConfig] each already
// check for via [buildFromNode]'s own type assertion when they resolve a
// nested "observer" entry. Splitting the registry in two would only
// duplicate that check earlier, for no added safety.
var observerRegistry = map[string]buildEntry{}

// RegisterObserver makes an observer type available for use as a nested
// "observer" entry of a drawer or reporter (see [RowObserverConfig],
// [MatrixObserverConfig], [Register]).
//
// For an observer that needs more than its own exported fields decoded
// directly, use [RegisterObserverFunc] instead.
func RegisterObserver[T any, PT interface{ *T }]() {
	register(observerRegistry, func(c T) PT { return PT(&c) })
}

// RegisterObserverFunc makes an observer type available (see
// [RegisterFunc], [RegisterObserver]).
func RegisterObserverFunc[C any, T any](build func(C) T) {
	register(observerRegistry, build)
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
// "sys"), this yields "pwn.sys.InitTrees". t may be a pointer type (as it
// usually is here, since a type is normally registered by its
// interface-implementing pointer); the pointer is transparently unwrapped
// first.
func qualifiedName(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

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
