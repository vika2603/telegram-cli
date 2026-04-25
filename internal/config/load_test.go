package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeTemp(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(p, []byte(body), 0600))
	return p
}

func TestLoad_minimal(t *testing.T) {
	p := writeTemp(t, `version = 1
default_account = "alice"`)
	var warn []string
	got, err := Load(p, func(s string) { warn = append(warn, s) })
	require.NoError(t, err)
	require.Equal(t, "alice", *got.DefaultAccount)
	require.Empty(t, warn)
}

func TestLoad_missingFile_returnsDefaults(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "nonexistent.toml"), func(string) {})
	require.NoError(t, err)
	require.Nil(t, got.DefaultAccount)
}

func TestLoad_wrongVersion_fatal(t *testing.T) {
	p := writeTemp(t, `version = 2`)
	_, err := Load(p, func(string) {})
	require.Error(t, err)
	require.Contains(t, err.Error(), "version")
}

func TestLoad_noVersion_fatal(t *testing.T) {
	p := writeTemp(t, `default_account = "x"`)
	_, err := Load(p, func(string) {})
	require.Error(t, err)
}

func TestLoad_unknownKey_warn(t *testing.T) {
	p := writeTemp(t, `version = 1
mystery_key = "x"`)
	var warn []string
	_, err := Load(p, func(s string) { warn = append(warn, s) })
	require.NoError(t, err)
	require.NotEmpty(t, warn)
}

func TestLoad_unknownNestedKey_warn(t *testing.T) {
	p := writeTemp(t, `version = 1

[output]
format = "json"
mystery = "x"`)
	var warn []string
	_, err := Load(p, func(s string) { warn = append(warn, s) })
	require.NoError(t, err)
	require.Len(t, warn, 1)
	require.Contains(t, warn[0], "mystery")
	require.Contains(t, warn[0], "[output]")
}

func TestLoad_aliasesTableNeverWarns(t *testing.T) {
	p := writeTemp(t, `version = 1

[aliases]
anything_the_user_wants = "me"`)
	var warn []string
	_, err := Load(p, func(s string) { warn = append(warn, s) })
	require.NoError(t, err)
	require.Empty(t, warn)
}
