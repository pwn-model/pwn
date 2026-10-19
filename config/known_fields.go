package config

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// checkEntryFields checks a "type" + parameters entry's node against the
// config shape t it is about to be decoded into, and reports all problems
// together as a *yaml.TypeError with the same message format as
// [yaml.Decoder.KnownFields] produces, or nil if there are none:
//
//   - every mapping key in node (recursively) that doesn't correspond to a
//     field of t, except "type" at node's own top level (it selects a
//     registry entry instead of being decoded into it);
//   - if required is set, every field of t without a key in node (see
//     [collectMissingFields]).
//
// This is needed because every "type" + parameters entry is decoded via
// [yaml.Node.Decode], which always decodes non-strictly: unlike
// [yaml.Decoder], it has no KnownFields option, and a strict top-level
// decoder doesn't pass its strictness on to custom UnmarshalYAML methods.
// Checking the original node here, instead of re-encoding it for a strict
// decoder, keeps the reported line numbers pointing into the config file.
//
// Field naming mirrors yaml's own struct field resolution (tag name, else
// the lowercased field name; "-" skips a field; ",inline" structs and maps
// are flattened). Recursion stops at types with their own UnmarshalYAML
// (e.g. [RowObserverConfig]), which check their own nodes when decoded.
func checkEntryFields(node *yaml.Node, t reflect.Type, required bool) error {
	var msgs []string
	collectUnknownFields(node, t, []string{"type"}, &msgs)
	if required {
		collectMissingFields(node, t, &msgs)
	}
	if len(msgs) > 0 {
		return &yaml.TypeError{Errors: msgs}
	}
	return nil
}

var unmarshalerType = reflect.TypeFor[yaml.Unmarshaler]()

func collectUnknownFields(node *yaml.Node, t reflect.Type, allowed []string, msgs *[]string) {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if node == nil {
		return
	}
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) > 0 {
			collectUnknownFields(node.Content[0], t, allowed, msgs)
		}
		return
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Implements(unmarshalerType) || reflect.PointerTo(t).Implements(unmarshalerType) {
		return
	}

	switch t.Kind() {
	case reflect.Struct:
		if node.Kind != yaml.MappingNode {
			return // A kind mismatch; left for the actual decode to report.
		}
		fields := map[string]reflect.Type{}
		anyKey := structFields(t, fields)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			if key.Kind == yaml.ScalarNode && key.ShortTag() == "!!merge" {
				// A "<<: *anchor" merge key: the merged mapping(s) are
				// decoded into the same struct.
				if value.Kind == yaml.SequenceNode {
					for _, v := range value.Content {
						collectUnknownFields(v, t, allowed, msgs)
					}
				} else {
					collectUnknownFields(value, t, allowed, msgs)
				}
				continue
			}
			if slices.Contains(allowed, key.Value) {
				continue
			}
			if ft, ok := fields[key.Value]; ok {
				collectUnknownFields(value, ft, nil, msgs)
			} else if !anyKey {
				*msgs = append(*msgs, fmt.Sprintf("line %d: field %s not found in type %s", key.Line, key.Value, t))
			}
		}
	case reflect.Slice, reflect.Array:
		if node.Kind == yaml.SequenceNode {
			for _, v := range node.Content {
				collectUnknownFields(v, t.Elem(), nil, msgs)
			}
		}
	case reflect.Map:
		if node.Kind == yaml.MappingNode {
			for i := 1; i < len(node.Content); i += 2 {
				collectUnknownFields(node.Content[i], t.Elem(), nil, msgs)
			}
		}
	}
}

// collectMissingFields reports every field of struct type t that has no key
// in the mapping node, naming the node's line. A field is required unless
// tagged `config:"optional"`, so that a parameter left out of a config file
// is an error instead of silently taking its zero value.
//
// Only node's own keys are checked, not those of nested values: nested
// "type" + parameters entries (e.g. an observer) check their own nodes when
// decoded.
func collectMissingFields(node *yaml.Node, t reflect.Type, msgs *[]string) {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if node == nil || node.Kind != yaml.MappingNode || t.Kind() != reflect.Struct {
		return // A kind mismatch; left for the actual decode to report.
	}

	keys := map[string]bool{}
	mappingKeys(node, keys)

	var missing []string
	for _, name := range requiredFields(t) {
		if !keys[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		*msgs = append(*msgs, fmt.Sprintf("line %d: missing field(s) %s in type %s", node.Line, strings.Join(missing, ", "), t))
	}
}

// mappingKeys adds the keys of mapping node to keys, including those of
// mappings merged in via "<<: *anchor".
func mappingKeys(node *yaml.Node, keys map[string]bool) {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if node == nil || node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Kind == yaml.ScalarNode && key.ShortTag() == "!!merge" {
			if value.Kind == yaml.SequenceNode {
				for _, v := range value.Content {
					mappingKeys(v, keys)
				}
			} else {
				mappingKeys(value, keys)
			}
			continue
		}
		keys[key.Value] = true
	}
}

// requiredFields returns the YAML keys of struct type t's fields that are
// not tagged `config:"optional"`, in field order. Field naming follows
// [structFields]; fields of ",inline" structs are included, while an
// ",inline" map or type with its own UnmarshalYAML requires nothing.
func requiredFields(t reflect.Type) []string {
	var names []string
	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue // Unexported.
		}
		if field.Tag.Get("config") == "optional" {
			continue
		}

		name, inline, skip := yamlFieldName(field)
		if skip {
			continue
		}
		if inline {
			ft := field.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && !reflect.PointerTo(ft).Implements(unmarshalerType) {
				names = append(names, requiredFields(ft)...)
			}
			continue
		}
		names = append(names, name)
	}
	return names
}

// yamlFieldName returns the YAML key of a struct field, as yaml itself
// resolves it (tag name, else the lowercased field name), whether the
// field is ",inline", and whether it is skipped via a "-" tag.
func yamlFieldName(field reflect.StructField) (name string, inline bool, skip bool) {
	tag := field.Tag.Get("yaml")
	if tag == "" && !strings.Contains(string(field.Tag), ":") {
		tag = string(field.Tag)
	}
	if tag == "-" {
		return "", false, true
	}
	name, flags, _ := strings.Cut(tag, ",")
	if slices.Contains(strings.Split(flags, ","), "inline") {
		return "", true, false
	}
	if name == "" {
		name = strings.ToLower(field.Name)
	}
	return name, false, false
}

// structFields adds the YAML keys of struct type t's fields, and the type
// each decodes into, to fields. It reports whether t has an ",inline" map
// field, which accepts any otherwise unknown key.
func structFields(t reflect.Type, fields map[string]reflect.Type) (anyKey bool) {
	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue // Unexported.
		}

		name, inline, skip := yamlFieldName(field)
		if skip {
			continue
		}

		if inline {
			ft := field.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			switch {
			case ft.Kind() == reflect.Map:
				anyKey = true
			case ft.Kind() == reflect.Struct && !reflect.PointerTo(ft).Implements(unmarshalerType):
				if structFields(ft, fields) {
					anyKey = true
				}
			default:
				// An inline type with its own UnmarshalYAML sees the
				// whole mapping; nothing can be checked here.
				anyKey = true
			}
			continue
		}

		fields[name] = field.Type
	}
	return anyKey
}
