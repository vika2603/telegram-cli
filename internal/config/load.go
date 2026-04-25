package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Warner is called for each unknown key encountered during Load. Warnings
// are written directly to stderr — they must not be silenceable via log
// level, so callers wire a plain os.Stderr writer, not zap.
type Warner func(msg string)

// Load reads a TOML config file. A missing file returns a zero Config and
// no error (defaults apply via Merge). A present file must have version = 1.
// Unknown keys at the top level or inside `[output]`, `[log]`, or
// `[flood_wait]` produce a stderr warning. The `[aliases]` table is
// user-defined and never warns.
func Load(path string, warn Warner) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var c Config
	if err := toml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("%w: parse config %s: %w", ErrInvalid, path, err)
	}
	if c.Version == nil {
		return Config{}, fmt.Errorf("%w: config %s: missing required 'version' field (expected 1)", ErrInvalid, path)
	}
	if *c.Version != 1 {
		return Config{}, fmt.Errorf("%w: config %s: version %d not supported by this tg binary, expected 1", ErrInvalid, path, *c.Version)
	}

	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err == nil {
		for k, v := range raw {
			if !isKnownTopLevel(k) {
				warn(fmt.Sprintf("config %s: unknown key %q (ignored)", path, k))
				continue
			}
			if nested, known := knownNestedKeys[k]; known {
				sub, ok := v.(map[string]any)
				if !ok {
					continue
				}
				for nk := range sub {
					if !nested[nk] {
						warn(fmt.Sprintf("config %s: unknown key %q in [%s] (ignored)", path, nk, k))
					}
				}
			}
		}
	}
	return c, nil
}

func isKnownTopLevel(k string) bool {
	switch k {
	case "version", "default_account", "api_id", "api_hash",
		"output", "log", "flood_wait", "aliases":
		return true
	}
	return false
}

// knownNestedKeys maps a section name to its allowed field set. `aliases`
// is omitted because it is user-defined.
var knownNestedKeys = map[string]map[string]bool{
	"output":     {"format": true, "color": true},
	"log":        {"level": true, "file": true, "format": true},
	"flood_wait": {"mode": true, "max_seconds": true},
}
