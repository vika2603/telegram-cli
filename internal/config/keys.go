package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Key is the canonical dotted-path identifier for a config value. Keys are
// closed-world: unknown keys fail validation. Each key has a type and, for
// enum keys, the permitted value set.
type Key struct {
	Name       string
	Type       KeyType
	EnumValues []string // populated only when Type == KeyEnum
}

// KeyType classifies the storage type of a config key.
type KeyType int

const (
	// KeyString holds an arbitrary string value.
	KeyString KeyType = iota
	// KeyInt holds a 64-bit integer value.
	KeyInt
	// KeyEnum holds one of a fixed set of string values.
	KeyEnum
	// KeyAlias is a dynamic aliases.<subkey>; value is an opaque string.
	KeyAlias
)

// Keys is the ordered list of every valid static dotted path. Order matters
// for help output and shell completion.
var Keys = []Key{
	{Name: "default_account", Type: KeyString},
	{Name: "api_id", Type: KeyInt},
	{Name: "api_hash", Type: KeyString},
	{Name: "output.format", Type: KeyEnum, EnumValues: []string{"human", "json"}},
	{Name: "output.color", Type: KeyEnum, EnumValues: []string{"auto", "always", "never"}},
	{Name: "log.level", Type: KeyEnum, EnumValues: []string{"error", "warn", "info", "debug"}},
	{Name: "log.file", Type: KeyString},
	{Name: "log.format", Type: KeyEnum, EnumValues: []string{"console", "json"}},
	{Name: "flood_wait.mode", Type: KeyEnum, EnumValues: []string{"fail", "wait"}},
	{Name: "flood_wait.max_seconds", Type: KeyInt},
	// aliases.<name> handled via prefix matching in ResolveKey.
}

// ResolveKey returns the Key matching name. For aliases.<anything> it returns
// a synthesized Key{Type: KeyAlias, Name: name}. An unknown key returns an
// error listing all valid keys.
func ResolveKey(name string) (Key, error) {
	if strings.HasPrefix(name, "aliases.") {
		sub := strings.TrimPrefix(name, "aliases.")
		if sub == "" {
			return Key{}, errors.New("aliases.<name>: subkey is required")
		}
		return Key{Name: name, Type: KeyAlias}, nil
	}
	for _, k := range Keys {
		if k.Name == name {
			return k, nil
		}
	}
	valid := make([]string, 0, len(Keys)+1)
	for _, k := range Keys {
		valid = append(valid, k.Name)
	}
	valid = append(valid, "aliases.<name>")
	return Key{}, fmt.Errorf("unknown config key %q; valid keys: %s", name, strings.Join(valid, ", "))
}

// CoerceValue parses a raw string value into the correct TOML-compatible Go
// type for k. Validation (enum membership, int range) is performed here.
func CoerceValue(k Key, raw string) (any, error) {
	switch k.Type {
	case KeyString, KeyAlias:
		return raw, nil
	case KeyInt:
		n, err := strconv.Atoi(raw)
		if err != nil {
			return nil, fmt.Errorf("%s: not an integer: %s", k.Name, err.Error())
		}
		if k.Name == "flood_wait.max_seconds" && n < 0 {
			return nil, fmt.Errorf("%s: must be non-negative", k.Name)
		}
		if k.Name == "api_id" && n <= 0 {
			return nil, fmt.Errorf("%s: must be positive", k.Name)
		}
		return int64(n), nil // TOML writer wants int64
	case KeyEnum:
		for _, ev := range k.EnumValues {
			if raw == ev {
				return raw, nil
			}
		}
		return nil, fmt.Errorf("%s: must be one of %s", k.Name, strings.Join(k.EnumValues, "|"))
	default:
		return nil, fmt.Errorf("%s: unknown key type", k.Name)
	}
}

// ReadRaw traverses raw (a decoded TOML map) along the dotted path name and
// returns the leaf value and whether it was present. The value type is
// whatever TOML decoded it as (string, int64, bool, map[string]any, etc.).
func ReadRaw(raw map[string]any, name string) (any, bool) {
	parts := strings.Split(name, ".")
	var cur any = raw
	for _, p := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, present := m[p]
		if !present {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// WriteRaw mutates raw so that the dotted-path name holds value. Missing
// intermediate maps are created automatically.
func WriteRaw(raw map[string]any, name string, value any) {
	parts := strings.Split(name, ".")
	m := raw
	for _, p := range parts[:len(parts)-1] {
		child, ok := m[p].(map[string]any)
		if !ok {
			child = map[string]any{}
			m[p] = child
		}
		m = child
	}
	m[parts[len(parts)-1]] = value
}

// UnsetRaw removes the key at the dotted path. Returns true if the key was
// present and removed, false if it did not exist. Empty intermediate maps
// created by a previous WriteRaw are not pruned.
func UnsetRaw(raw map[string]any, name string) bool {
	parts := strings.Split(name, ".")
	m := raw
	for _, p := range parts[:len(parts)-1] {
		child, ok := m[p].(map[string]any)
		if !ok {
			return false
		}
		m = child
	}
	last := parts[len(parts)-1]
	if _, ok := m[last]; !ok {
		return false
	}
	delete(m, last)
	return true
}
