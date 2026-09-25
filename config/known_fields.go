package config

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	yaml "go.yaml.in/yaml/v3"
)

// checkKnownFields reports every mapping key in node (recursively) that
// doesn't correspond to a field of the Go type it is about to be decoded
// into, as a *yaml.TypeError with the same message format as
// [yaml.Decoder.KnownFields] produces, or nil if there are none. Keys in
// allowed are accepted at node's own top level only (e.g. "type", which
// selects a registry entry instead of being decoded into it).
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
func checkKnownFields(node *yaml.Node, t reflect.Type, allowed ...string) error {
	var msgs []string
	collectUnknownFields(node, t, allowed, &msgs)
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

// structFields adds the YAML keys of struct type t's fields, and the type
// each decodes into, to fields. It reports whether t has an ",inline" map
// field, which accepts any otherwise unknown key.
func structFields(t reflect.Type, fields map[string]reflect.Type) (anyKey bool) {
	for i := range t.NumField() {
		field := t.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue // Unexported.
		}

		tag := field.Tag.Get("yaml")
		if tag == "" && !strings.Contains(string(field.Tag), ":") {
			tag = string(field.Tag)
		}
		if tag == "-" {
			continue
		}
		name, flags, _ := strings.Cut(tag, ",")

		if slices.Contains(strings.Split(flags, ","), "inline") {
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

		if name == "" {
			name = strings.ToLower(field.Name)
		}
		fields[name] = field.Type
	}
	return anyKey
}
