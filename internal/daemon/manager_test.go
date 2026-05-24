package daemon_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/daemon"
)

func TestResolve_FillsDefaults(t *testing.T) {
	cfg := daemon.Config{Account: "alice"}
	require.NoError(t, daemon.Resolve(&cfg))

	require.NotEmpty(t, cfg.BinaryPath, "BinaryPath should default to current executable")
	require.NotEmpty(t, cfg.LogFile, "LogFile should default to per-account daemon dir")
	require.Equal(t, int64(daemon.DefaultLogMaxSize), cfg.LogMaxSize)
}

func TestResolve_PreservesExplicitValues(t *testing.T) {
	cfg := daemon.Config{
		Account:    "alice",
		BinaryPath: "/custom/path/tg",
		LogFile:    "/custom/log",
		LogMaxSize: 1234,
	}
	require.NoError(t, daemon.Resolve(&cfg))
	require.Equal(t, "/custom/path/tg", cfg.BinaryPath)
	require.Equal(t, "/custom/log", cfg.LogFile)
	require.Equal(t, int64(1234), cfg.LogMaxSize)
}

func TestResolve_RejectsEmptyAccount(t *testing.T) {
	err := daemon.Resolve(&daemon.Config{})
	require.Error(t, err)
}

func TestNewManager_RejectsEmptyAccount(t *testing.T) {
	_, err := daemon.NewManager("")
	require.Error(t, err)
}

func TestSaveLoadRemoveMeta_RoundTrip(t *testing.T) {
	tempXDG := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempXDG)

	m := &daemon.Meta{
		Account:     "alice",
		LogFile:     filepath.Join(tempXDG, "tg/accounts/alice/daemon/daemon.log"),
		LogMaxSize:  daemon.DefaultLogMaxSize,
		BinaryPath:  "/tmp/tg",
		InstalledAt: "2026-05-25T00:00:00Z",
		Platform:    "launchd",
	}
	require.NoError(t, daemon.SaveMeta(m))

	got, err := daemon.LoadMeta("alice")
	require.NoError(t, err)
	require.Equal(t, m.Account, got.Account)
	require.Equal(t, m.LogFile, got.LogFile)
	require.Equal(t, m.Platform, got.Platform)

	require.NoError(t, daemon.RemoveMeta("alice"))

	_, err = daemon.LoadMeta("alice")
	require.ErrorIs(t, err, os.ErrNotExist, "LoadMeta after RemoveMeta should be os.ErrNotExist")
}

func TestRemoveMeta_IsIdempotent(t *testing.T) {
	tempXDG := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempXDG)
	require.NoError(t, daemon.RemoveMeta("nonexistent"))
}

func TestSaveMeta_RequiresAccount(t *testing.T) {
	require.Error(t, daemon.SaveMeta(&daemon.Meta{}))
}

func TestPaths_AnchoredOnXDGConfigHome(t *testing.T) {
	tempXDG := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempXDG)
	expected := filepath.Join(tempXDG, "tg", "accounts", "alice", "daemon")
	require.Equal(t, expected, daemon.DaemonDir("alice"))
	require.Equal(t, filepath.Join(expected, "daemon.json"), daemon.MetaFile("alice"))
	require.Equal(t, filepath.Join(expected, "daemon.log"), daemon.LogFile("alice"))
	require.Equal(t, filepath.Join(expected, "updates.ndjson"), daemon.UpdatesFile("alice"))
	require.Equal(t, filepath.Join(expected, "daemon.sock"), daemon.SocketPath("alice"))
}
