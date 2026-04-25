package config

import (
	"github.com/pelletier/go-toml/v2"

	"github.com/vika2603/telegram-cli/internal/account"
)

// MergedConfig is the result of the full precedence pipeline. It carries the
// typed merged Config, the resolved file path, the raw TOML map (used for
// source-aware reads by config get), and a Sources map that records which
// precedence layer supplied each key.
type MergedConfig struct {
	Config  Config
	Path    string
	Raw     map[string]any
	Sources map[string]string
}

// LoadMerged runs the full precedence pipeline: defaults < file < env <
// flagCfg, then enum-validates the result. flagConfigPath is the --config
// value (empty = default path). The returned path is the resolved config
// file path, useful for `config path` and warning diagnostics.
//
// LoadMerged is a thin wrapper around LoadMergedWithSources.
func LoadMerged(flagCfg Config, flagConfigPath string) (Config, string, error) {
	m, err := LoadMergedWithSources(flagCfg, flagConfigPath)
	if err != nil {
		return Config{}, m.Path, err
	}
	return m.Config, m.Path, nil
}

// LoadMergedWithSources is the canonical implementation of the precedence
// pipeline. It returns a MergedConfig whose Sources map records the layer
// ("default", "file", "env", "flag") that last set each key.
//
// Precedence (lowest to highest): Defaults < File < Env < flagCfg.
//
// The flagCfg argument carries real flag-layer config values (e.g. a future
// flag that sets output.format). Root runtime flags like --account and
// --output are selector/formatter controls and must NOT be passed here;
// callers should pass config.Config{} when only reading the config.
func LoadMergedWithSources(flagCfg Config, flagConfigPath string) (MergedConfig, error) {
	path := ResolvePath(flagConfigPath, account.ConfigFile())

	// --- layer 1: Defaults ---
	defCfg := Defaults()
	defRaw, err := configToRawMap(defCfg)
	if err != nil {
		return MergedConfig{Path: path}, err
	}

	// --- layer 2: File ---
	fileCfg, err := Load(path, func(string) {})
	if err != nil {
		return MergedConfig{Path: path}, err
	}
	fileRaw, err := configToRawMap(fileCfg)
	if err != nil {
		return MergedConfig{Path: path}, err
	}
	// aliases are a map (not pointer), so they are always present in fileRaw
	// even when the file has no [aliases] section. Strip them unless the file
	// actually set at least one alias.
	if len(fileCfg.Aliases) == 0 {
		delete(fileRaw, "aliases")
	}

	// --- layer 3: Env ---
	envCfg, err := FromEnv()
	if err != nil {
		return MergedConfig{Path: path}, err
	}
	envRaw, err := configToRawMap(envCfg)
	if err != nil {
		return MergedConfig{Path: path}, err
	}

	// --- layer 4: Flag ---
	flagRaw, err := configToRawMap(flagCfg)
	if err != nil {
		return MergedConfig{Path: path}, err
	}

	// Build the merged typed Config via the existing Merge chain.
	merged := Merge(defCfg, fileCfg)
	merged = Merge(merged, envCfg)
	merged = Merge(merged, flagCfg)
	if err := ValidateEnums(merged); err != nil {
		return MergedConfig{Path: path}, err
	}

	// Build the merged raw map and track sources layer by layer.
	// Later layers overwrite earlier ones only when they actually set a key.
	mergedRaw := map[string]any{}
	sources := map[string]string{}

	applyLayer := func(layerRaw map[string]any, source string) {
		applyRawLayer(mergedRaw, sources, layerRaw, source, "")
	}
	applyLayer(defRaw, "default")
	applyLayer(fileRaw, "file")
	applyLayer(envRaw, "env")
	applyLayer(flagRaw, "flag")

	return MergedConfig{
		Config:  merged,
		Path:    path,
		Raw:     mergedRaw,
		Sources: sources,
	}, nil
}

// applyRawLayer walks src recursively and writes non-nil scalar values into
// dst, recording source for each dotted key. prefix is the dotted-path built
// so far; empty at the top level.
func applyRawLayer(dst map[string]any, sources map[string]string, src map[string]any, source, prefix string) {
	for k, v := range src {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if sub, ok := v.(map[string]any); ok {
			// Recurse into nested tables (output, log, flood_wait, aliases).
			child, ok2 := dst[k].(map[string]any)
			if !ok2 {
				child = map[string]any{}
				dst[k] = child
			}
			applyRawLayer(child, sources, sub, source, key)
		} else if v != nil {
			dst[k] = v
			sources[key] = source
		}
	}
}

// configToRawMap marshals c through go-toml then unmarshals it back into
// map[string]any. This collapses all pointer-indirection and struct nesting
// into the same shape that Load produces from a TOML file. Nil pointer fields
// do not contribute any key in the resulting map.
func configToRawMap(c Config) (map[string]any, error) {
	b, err := toml.Marshal(c)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := toml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}
