package daemon_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/daemon"
)

func TestRotateIfLarger_NoFileIsNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.log")
	require.NoError(t, daemon.RotateIfLarger(path, 100))
	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err))
}

func TestRotateIfLarger_BelowThresholdLeavesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log")
	require.NoError(t, os.WriteFile(path, []byte("hi"), 0o600))
	require.NoError(t, daemon.RotateIfLarger(path, 100))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "hi", string(data))
	_, err = os.Stat(path + ".1")
	require.True(t, os.IsNotExist(err))
}

func TestRotateIfLarger_AboveThresholdMovesToBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log")
	require.NoError(t, os.WriteFile(path, []byte("this is too long"), 0o600))
	require.NoError(t, daemon.RotateIfLarger(path, 8))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Empty(t, data, "fresh file should be empty after rotation")

	backup, err := os.ReadFile(path + ".1")
	require.NoError(t, err)
	require.Equal(t, "this is too long", string(backup))
}

func TestRotateIfLarger_OverwritesExistingBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log")
	require.NoError(t, os.WriteFile(path, []byte("newer"), 0o600))
	require.NoError(t, os.WriteFile(path+".1", []byte("older"), 0o600))
	require.NoError(t, daemon.RotateIfLarger(path, 1))

	backup, err := os.ReadFile(path + ".1")
	require.NoError(t, err)
	require.Equal(t, "newer", string(backup))
}

func TestRotateIfLarger_ZeroDisablesRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log")
	require.NoError(t, os.WriteFile(path, []byte("anything"), 0o600))
	require.NoError(t, daemon.RotateIfLarger(path, 0))
	_, err := os.Stat(path + ".1")
	require.True(t, os.IsNotExist(err))
}

func TestEnsureDir_CreatesNestedPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")
	require.NoError(t, daemon.EnsureDir(target))
	info, err := os.Stat(target)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}
