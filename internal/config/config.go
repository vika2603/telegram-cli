// Package config defines the Config struct, its TOML schema, merge semantics,
// environment-variable sparse loader, and file loader.
// All fields are pointers or explicit sentinel-empty (maps). A nil pointer means
// "this layer did not set this field", allowing Merge to only overwrite with
// values that were actually provided.
package config

type Config struct {
	Version        *int              `toml:"version"`
	DefaultAccount *string           `toml:"default_account"`
	APIID          *int              `toml:"api_id"`
	APIHash        *string           `toml:"api_hash"`
	Output         OutputCfg         `toml:"output"`
	Log            LogCfg            `toml:"log"`
	FloodWait      FloodWaitCfg      `toml:"flood_wait"`
	Aliases        map[string]string `toml:"aliases"`
}

type OutputCfg struct {
	Format *string `toml:"format" json:"format,omitempty"` // "human" | "json"
	Color  *string `toml:"color"  json:"color,omitempty"`  // "auto" | "always" | "never"
}

type LogCfg struct {
	Level  *string `toml:"level"  json:"level,omitempty"`  // "error" | "warn" | "info" | "debug"
	File   *string `toml:"file"   json:"file,omitempty"`   // empty = stderr
	Format *string `toml:"format" json:"format,omitempty"` // "console" | "json"
}

type FloodWaitCfg struct {
	Mode       *string `toml:"mode"        json:"mode,omitempty"`        // "fail" | "wait"
	MaxSeconds *int    `toml:"max_seconds" json:"max_seconds,omitempty"` // 0 = unlimited
}

// Defaults returns the hard-coded baseline config — the lowest-precedence
// layer under env and the file-backed config.
func Defaults() Config {
	return Config{
		Output:    OutputCfg{Format: ptr("human"), Color: ptr("auto")},
		Log:       LogCfg{Level: ptr("warn"), File: ptr(""), Format: ptr("console")},
		FloodWait: FloodWaitCfg{Mode: ptr("fail"), MaxSeconds: ptr(30)},
	}
}

// ptr returns a pointer to v. Generic helper so Defaults() and tests can
// build sparse Config values without a per-type wrapper.
func ptr[T any](v T) *T { return &v }
