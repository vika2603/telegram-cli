package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultAccount_writesNewFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	require.NoError(t, SetDefaultAccount(p, "alice"))
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, toml.Unmarshal(data, &raw))
	require.Equal(t, int64(1), raw["version"])
	require.Equal(t, "alice", raw["default_account"])
}

func TestSetDefaultAccount_preservesUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(p, []byte(`version = 1
default_account = "bob"
some_future_key = "keep-me"

[output]
format = "human"
`), 0600))
	require.NoError(t, SetDefaultAccount(p, "alice"))
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, toml.Unmarshal(data, &raw))
	require.Equal(t, "alice", raw["default_account"])
	require.Equal(t, "keep-me", raw["some_future_key"])
	out := raw["output"].(map[string]any)
	require.Equal(t, "human", out["format"])
}

func TestSetDefaultAccount_permissions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "nested", "config.toml")
	require.NoError(t, SetDefaultAccount(p, "alice"))
	info, err := os.Stat(p)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestUnsetDefaultAccount_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	// Write a config that has default_account plus other keys that must survive.
	require.NoError(t, os.WriteFile(p, []byte(`version = 1
api_id = 42
default_account = "work"
`), 0600))
	require.NoError(t, UnsetDefaultAccount(p))
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, toml.Unmarshal(data, &raw))
	_, hasKey := raw["default_account"]
	require.False(t, hasKey, "default_account key should be absent after unset")
	require.Equal(t, int64(1), raw["version"])
	require.Equal(t, int64(42), raw["api_id"])
}

func TestUnsetDefaultAccount_KeyAbsent_NoOp(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(p, []byte(`version = 1
api_id = 7
`), 0600))
	require.NoError(t, UnsetDefaultAccount(p))
	data, err := os.ReadFile(p)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, toml.Unmarshal(data, &raw))
	_, hasKey := raw["default_account"]
	require.False(t, hasKey)
	require.Equal(t, int64(7), raw["api_id"])
}

func TestUnsetDefaultAccount_MissingFile_NoOp(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "does-not-exist.toml")
	require.NoError(t, UnsetDefaultAccount(p))
	// File should still not exist.
	_, err := os.Stat(p)
	require.True(t, os.IsNotExist(err))
}

func TestResolvePath_flagBeatsEnvBeatsDefault(t *testing.T) {
	t.Setenv("TG_CONFIG", "/env/cfg")
	got := ResolvePath("/flag/cfg", "/default/cfg")
	require.Equal(t, "/flag/cfg", got)

	got = ResolvePath("", "/default/cfg")
	require.Equal(t, "/env/cfg", got)

	t.Setenv("TG_CONFIG", "")
	got = ResolvePath("", "/default/cfg")
	require.Equal(t, "/default/cfg", got)
}
