package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// clearTGEnv unsets every TG_* environment variable that FromEnv reads so
// tests that call LoadMergedWithSources start from a clean slate. The
// variables are fully unset (not set to ""), because TG_LOG_FILE="" is
// meaningful in FromEnv (it overrides the default to "reset to stderr").
func clearTGEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"TG_ACCOUNT", "TG_API_ID", "TG_API_HASH", "TG_OUTPUT", "TG_COLOR",
		"TG_LOG_LEVEL", "TG_LOG_FORMAT", "TG_FLOOD_WAIT_MODE",
		"TG_FLOOD_WAIT_MAX", "TG_CONFIG",
	} {
		t.Setenv(k, "")
	}
	// TG_LOG_FILE must be fully unset rather than set to "" because FromEnv
	// treats the empty-string case as a meaningful override.
	orig, hadOrig := os.LookupEnv("TG_LOG_FILE")
	if err := os.Unsetenv("TG_LOG_FILE"); err != nil {
		t.Fatalf("clearTGEnv unsetenv TG_LOG_FILE: %v", err)
	}
	t.Cleanup(func() {
		if hadOrig {
			_ = os.Setenv("TG_LOG_FILE", orig)
		} else {
			_ = os.Unsetenv("TG_LOG_FILE")
		}
	})
}

// writeTempConfig creates a minimal TOML config file in a temp dir and
// returns its path.
func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(p, []byte(body), 0600))
	return p
}

func TestLoadMergedWithSources_DefaultOnly(t *testing.T) {
	clearTGEnv(t)
	// Use a path that does not exist so only defaults apply.
	path := filepath.Join(t.TempDir(), "nonexistent.toml")

	m, err := LoadMergedWithSources(Config{}, path)
	require.NoError(t, err)

	// Every key provided by Defaults should be sourced from "default".
	require.Equal(t, "default", m.Sources["output.format"])
	require.Equal(t, "default", m.Sources["output.color"])
	require.Equal(t, "default", m.Sources["log.level"])
	require.Equal(t, "default", m.Sources["log.file"])
	require.Equal(t, "default", m.Sources["log.format"])
	require.Equal(t, "default", m.Sources["flood_wait.mode"])
	require.Equal(t, "default", m.Sources["flood_wait.max_seconds"])
}

func TestLoadMergedWithSources_FileOverridesDefault(t *testing.T) {
	clearTGEnv(t)
	path := writeTempConfig(t, `version = 1
[output]
format = "json"`)

	m, err := LoadMergedWithSources(Config{}, path)
	require.NoError(t, err)

	require.Equal(t, "file", m.Sources["output.format"])
	// other output key still from default
	require.Equal(t, "default", m.Sources["output.color"])
}

func TestLoadMergedWithSources_EnvOverridesFile(t *testing.T) {
	clearTGEnv(t)
	path := writeTempConfig(t, `version = 1
[output]
format = "json"`)
	t.Setenv("TG_OUTPUT", "human")

	m, err := LoadMergedWithSources(Config{}, path)
	require.NoError(t, err)

	require.Equal(t, "env", m.Sources["output.format"])
}

func TestLoadMergedWithSources_FlagOverridesEnv(t *testing.T) {
	clearTGEnv(t)
	path := writeTempConfig(t, `version = 1`)
	t.Setenv("TG_OUTPUT", "human")

	flagCfg := Config{Output: OutputCfg{Format: ptr("json")}}
	m, err := LoadMergedWithSources(flagCfg, path)
	require.NoError(t, err)

	require.Equal(t, "flag", m.Sources["output.format"])
}

func TestLoadMergedWithSources_AliasesFromFile(t *testing.T) {
	clearTGEnv(t)
	path := writeTempConfig(t, `version = 1

[aliases]
boss = "@foo"`)

	m, err := LoadMergedWithSources(Config{}, path)
	require.NoError(t, err)

	require.Equal(t, "file", m.Sources["aliases.boss"])
	v, ok := ReadRaw(m.Raw, "aliases.boss")
	require.True(t, ok)
	require.Equal(t, "@foo", v)
}

func TestLoadMergedWithSources_LoadMergedWrapper(t *testing.T) {
	clearTGEnv(t)
	path := writeTempConfig(t, `version = 1
default_account = "alice"`)

	cfg, resolvedPath, err := LoadMerged(Config{}, path)
	require.NoError(t, err)
	require.Equal(t, path, resolvedPath)
	require.NotNil(t, cfg.DefaultAccount)
	require.Equal(t, "alice", *cfg.DefaultAccount)
}

func TestLoadMergedWithSources_EnvLogFileEmpty(t *testing.T) {
	// TG_LOG_FILE="" is a meaningful env override (reset to stderr).
	// clearTGEnv unsets TG_LOG_FILE; here we set it to "" explicitly after.
	clearTGEnv(t)
	path := filepath.Join(t.TempDir(), "nonexistent.toml")
	// Set TG_LOG_FILE to the empty string — FromEnv treats this as an override
	// (distinct from the variable being absent).
	t.Setenv("TG_LOG_FILE", "")

	m, err := LoadMergedWithSources(Config{}, path)
	require.NoError(t, err)

	// TG_LOG_FILE="" is meaningful — it should come from env, not default.
	require.Equal(t, "env", m.Sources["log.file"])
}
