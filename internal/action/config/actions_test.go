package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
	appconfig "github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0600))
	return path
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"TG_ACCOUNT", "TG_API_ID", "TG_API_HASH", "TG_OUTPUT", "TG_COLOR",
		"TG_LOG_LEVEL", "TG_LOG_FORMAT", "TG_FLOOD_WAIT_MODE",
		"TG_FLOOD_WAIT_MAX", "TG_CONFIG",
	} {
		t.Setenv(key, "")
	}
	orig, hadOrig := os.LookupEnv("TG_LOG_FILE")
	require.NoError(t, os.Unsetenv("TG_LOG_FILE"))
	t.Cleanup(func() {
		if hadOrig {
			_ = os.Setenv("TG_LOG_FILE", orig)
			return
		}
		_ = os.Unsetenv("TG_LOG_FILE")
	})
}

func TestGetRedactsAPIHash(t *testing.T) {
	clearEnv(t)
	path := writeConfig(t, "version = 1\napi_hash = \"abcdefghijklmnopqrstuvwxyz\"\n")

	row, err := Get(context.Background(), GetRequest{Key: "api_hash", ConfigPath: path})
	require.NoError(t, err)
	require.Equal(t, "api_hash", row.Key)
	require.Equal(t, "abcdefgh…****", row.Value)
}

func TestSetWritesRealAPIHashButReturnsRedactedRow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")

	row, err := Set(context.Background(), SetRequest{
		Key:        "api_hash",
		Value:      "abcdefghijklmnopqrstuvwxyz123456",
		ConfigPath: path,
		Force:      true,
		Yes:        true,
		Prompter:   &ui.StubPrompter{},
	})
	require.NoError(t, err)
	require.Equal(t, "abcdefgh…****", row.New)

	raw, err := appconfig.ReadRawAt(path)
	require.NoError(t, err)
	got, ok := appconfig.ReadRaw(raw, "api_hash")
	require.True(t, ok)
	require.Equal(t, "abcdefghijklmnopqrstuvwxyz123456", got)
}

func TestUnsetMissingKeyReturnsNoResults(t *testing.T) {
	path := writeConfig(t, "version = 1\n")

	_, err := Unset(context.Background(), UnsetRequest{Key: "output.format", ConfigPath: path})
	require.Error(t, err)
	var noResults *command.NoResultsError
	require.ErrorAs(t, err, &noResults)
}
