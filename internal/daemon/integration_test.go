package daemon_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/daemon"
)

// TestIntegration_MaybeDialDaemonHandshake stands up a real daemon
// Server on a per-account socket under a temp $XDG_CONFIG_HOME, then
// exercises the same client path the daemon-aware CLI commands take:
// DaemonReachable -> Dial -> Hello -> Call. A registered "echo"
// handler validates that application-level RPCs round-trip end-to-end.
//
// macOS limits sun_path to 104 bytes, so we anchor the temp root under
// /tmp directly (not t.TempDir, whose default path is too long).
func TestIntegration_MaybeDialDaemonHandshake(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "tgd-int")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Setenv("XDG_CONFIG_HOME", root)

	account := "alice"
	require.NoError(t, daemon.EnsureDir(daemon.DaemonDir(account)))

	subs := daemon.NewSubscriptionManager(4)
	defer subs.Close()
	srv := daemon.NewServer(account, daemon.SocketPath(account), subs, nil)
	require.NoError(t, srv.Listen())
	defer func() { _ = srv.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	doneCh := make(chan struct{})
	go func() { _ = srv.Serve(ctx); close(doneCh) }()

	// Application handler that echoes back the params verbatim.
	srv.Register("echo", func(_ context.Context, params json.RawMessage) (json.RawMessage, error) {
		return params, nil
	})

	require.True(t, daemon.DaemonReachable(account),
		"socket should be reachable after Listen")

	cl, err := daemon.Dial(context.Background(), account)
	require.NoError(t, err)
	defer func() { _ = cl.Close() }()

	hello := cl.Hello()
	require.Equal(t, account, hello.Account)
	require.Equal(t, daemon.ProtocolSchema, hello.Schema)

	raw, err := cl.Call(context.Background(), "echo", map[string]any{
		"kind":  "msg.list",
		"limit": 5,
	})
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	require.Equal(t, "msg.list", got["kind"])
	require.InDelta(t, 5, got["limit"], 0)

	// Tear down: cancel and confirm Serve exits cleanly.
	cancel()
	<-doneCh
}

// TestIntegration_DaemonReachable_FalseWithoutSocket guards the
// no-daemon branch: when the per-account directory exists but there
// is no socket, DaemonReachable must return false so callers fall
// back to the local MTProto path.
func TestIntegration_DaemonReachable_FalseWithoutSocket(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "tgd-int")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Setenv("XDG_CONFIG_HOME", root)

	account := "alice"
	require.NoError(t, os.MkdirAll(filepath.Dir(daemon.SocketPath(account)), 0o700))

	require.False(t, daemon.DaemonReachable(account),
		"reachable should be false when no socket exists")
}
