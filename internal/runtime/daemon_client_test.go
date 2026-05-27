package runtime_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// TestMaybeDialDaemon_WarnsOnStaleSocket guards the single
// stale-socket warning path: socket file is a real Unix socket
// (ModeSocket set) but no process is accepting, so net.DialTimeout
// fails inside the probe window.
//
// We bind a Unix listener, set SetUnlinkOnClose(false) so the socket
// file outlives the listener, then close the listener to stop
// accepting. The resulting file is exactly the post-crash residue the
// warning is designed for.
//
// macOS limits sun_path to 104 bytes, so we anchor the temp root
// under /tmp directly via os.MkdirTemp rather than t.TempDir.
func TestMaybeDialDaemon_WarnsOnStaleSocket(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "tgd-warn")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Setenv("XDG_CONFIG_HOME", root)

	acct := &account.Account{}
	acct.Meta.Name = "alice"

	sockPath := filepath.Join(root, "tg", "accounts", "alice", "daemon", "daemon.sock")
	require.NoError(t, os.MkdirAll(filepath.Dir(sockPath), 0o700))

	ln, err := net.Listen("unix", sockPath)
	require.NoError(t, err)
	ul, ok := ln.(*net.UnixListener)
	require.True(t, ok)
	ul.SetUnlinkOnClose(false) // keep the socket file after Close
	require.NoError(t, ul.Close())
	t.Cleanup(func() { _ = os.Remove(sockPath) })

	info, statErr := os.Stat(sockPath)
	require.NoError(t, statErr)
	require.NotZero(t, info.Mode()&os.ModeSocket, "test scaffold must leave a real socket file")

	ios, _, _, errBuf := ui.Test()
	inv := &runtime.Invocation{IOStreams: ios}

	cl, err := runtime.MaybeDialDaemon(context.Background(), inv, acct)
	require.NoError(t, err)
	require.Nil(t, cl, "stale socket must yield a nil client so the caller falls back")
	require.Contains(t, errBuf.String(), `warning: daemon socket for account "alice" exists but is unreachable`)
	require.Contains(t, errBuf.String(), "tg daemon status")
}

// TestMaybeDialDaemon_QuietWhenNotInstalled asserts that the routine
// "daemon never installed" path is silent. Stderr must stay empty.
func TestMaybeDialDaemon_QuietWhenNotInstalled(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "tgd-noinst")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Setenv("XDG_CONFIG_HOME", root)

	acct := &account.Account{}
	acct.Meta.Name = "alice"

	ios, _, _, errBuf := ui.Test()
	inv := &runtime.Invocation{IOStreams: ios}

	cl, err := runtime.MaybeDialDaemon(context.Background(), inv, acct)
	require.NoError(t, err)
	require.Nil(t, cl)
	require.Empty(t, errBuf.String(), "no-install path must be silent")
}

// TestMaybeDialDaemon_QuietWhenNoDaemonFlag asserts --no-daemon is
// also silent — user explicit opt-out should not produce noise.
func TestMaybeDialDaemon_QuietWhenNoDaemonFlag(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "tgd-no-daemon")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Setenv("XDG_CONFIG_HOME", root)

	acct := &account.Account{}
	acct.Meta.Name = "alice"

	ios, _, _, errBuf := ui.Test()
	inv := &runtime.Invocation{IOStreams: ios, NoDaemon: true}

	cl, err := runtime.MaybeDialDaemon(context.Background(), inv, acct)
	require.NoError(t, err)
	require.Nil(t, cl)
	require.Empty(t, errBuf.String(), "--no-daemon path must be silent")
}
