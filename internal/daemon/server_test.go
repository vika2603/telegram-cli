package daemon_test

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

// withServer spins up a real Server bound to a temp Unix socket and
// returns the wired SubscriptionManager + a teardown. macOS limits
// sun_path to 104 bytes, which `t.TempDir()` paths routinely exceed
// (the harness uses long $TMPDIR + test-name + counter directories),
// so we anchor the socket under /tmp directly.
func withServer(t *testing.T, resolver daemon.PeerRefResolver) (*daemon.SubscriptionManager, string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "tgd")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "d.sock")
	mgr := daemon.NewSubscriptionManager(8)
	srv := daemon.NewServer("alice", sock, mgr, resolver)
	require.NoError(t, srv.Listen())

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	go func() {
		_ = srv.Serve(ctx)
		close(doneCh)
	}()
	teardown := func() {
		cancel()
		_ = srv.Close()
		<-doneCh
		mgr.Close()
	}
	return mgr, sock, teardown
}

func dialClient(t *testing.T, sock string) *daemon.Client {
	t.Helper()
	// Override SocketPath dispatch by colocating the socket where Dial
	// looks for it. The Dial path resolves SocketPath(account); rather
	// than tampering with XDG inside this test, we connect via a
	// dedicated helper that knows the absolute path.
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(sock)))))
	// SocketPath is account-aware; we used a temp path that doesn't
	// fit that scheme, so dial directly with net.Dial here.
	conn, err := net.DialTimeout("unix", sock, time.Second)
	require.NoError(t, err)
	cl, err := daemon.AttachClient(conn)
	require.NoError(t, err)
	return cl
}

func TestServer_HelloFrameOnConnect(t *testing.T) {
	_, sock, teardown := withServer(t, nil)
	defer teardown()

	cl := dialClient(t, sock)
	defer func() { _ = cl.Close() }()

	hello := cl.Hello()
	require.Equal(t, "alice", hello.Account)
	require.Equal(t, daemon.ProtocolSchema, hello.Schema)
}

func TestServer_PingPong(t *testing.T) {
	_, sock, teardown := withServer(t, nil)
	defer teardown()

	cl := dialClient(t, sock)
	defer func() { _ = cl.Close() }()

	raw, err := cl.Call(context.Background(), "ping", nil)
	require.NoError(t, err)
	require.JSONEq(t, `"pong"`, string(raw))
}

func TestServer_SubscribePublishUnsubscribe(t *testing.T) {
	mgr, sock, teardown := withServer(t, nil)
	defer teardown()

	cl := dialClient(t, sock)
	defer func() { _ = cl.Close() }()

	subID, err := cl.SubscribeRaw(context.Background(), daemon.SubscribeParams{})
	require.NoError(t, err)
	require.NotZero(t, subID)

	// Wait until the subscription is registered on the server side
	// before publishing (Subscribe RPC returns before streamSubscription
	// has gone around the goroutine; publish too eagerly and the event
	// disappears into a not-yet-existing channel — actually no, the
	// channel exists from Subscribe time, but the goroutine that drains
	// it starts in streamSubscription. Either way Publish is safe).
	mgr.Publish(telegram.WatchEvent{Kind: telegram.EventNewMessage, Row: output.MessageRow{ID: 7, Text: "hi"}})

	select {
	case f, ok := <-cl.Events:
		require.True(t, ok)
		require.Equal(t, "update", f.Event)
		require.Equal(t, subID, f.Sub)
	case <-time.After(time.Second):
		t.Fatal("expected update event")
	}

	require.NoError(t, cl.Unsubscribe(context.Background(), subID))
}

func TestServer_SubscribeWithUnknownRefReturnsError(t *testing.T) {
	resolver := func(_ context.Context, _ string) (int64, error) {
		return 0, errors.New("not found")
	}
	_, sock, teardown := withServer(t, resolver)
	defer teardown()

	cl := dialClient(t, sock)
	defer func() { _ = cl.Close() }()

	_, err := cl.SubscribeRaw(context.Background(), daemon.SubscribeParams{
		Refs: []string{"@nonsense"},
	})
	require.Error(t, err)
	var rem *daemon.RemoteError
	require.ErrorAs(t, err, &rem)
	require.Equal(t, "resolve_failed", rem.Code)
}

func TestServer_RefsWithoutResolverIsErrored(t *testing.T) {
	_, sock, teardown := withServer(t, nil)
	defer teardown()

	cl := dialClient(t, sock)
	defer func() { _ = cl.Close() }()

	_, err := cl.SubscribeRaw(context.Background(), daemon.SubscribeParams{
		Refs: []string{"@chan"},
	})
	require.Error(t, err)
}

func TestServer_ConnectionDropCleansUpSubscriptions(t *testing.T) {
	mgr, sock, teardown := withServer(t, nil)
	defer teardown()

	cl := dialClient(t, sock)
	_, err := cl.SubscribeRaw(context.Background(), daemon.SubscribeParams{})
	require.NoError(t, err)
	require.Equal(t, 1, mgr.Len())

	_ = cl.Close()
	// Server-side cleanup is asynchronous; poll briefly.
	require.Eventually(t, func() bool { return mgr.Len() == 0 }, time.Second, 20*time.Millisecond)
}

func TestServer_UnknownMethodIsErrored(t *testing.T) {
	_, sock, teardown := withServer(t, nil)
	defer teardown()

	cl := dialClient(t, sock)
	defer func() { _ = cl.Close() }()

	_, err := cl.Call(context.Background(), "fly_to_the_moon", nil)
	require.Error(t, err)
	var rem *daemon.RemoteError
	require.ErrorAs(t, err, &rem)
	require.Equal(t, "unknown_method", rem.Code)
}

// TestServer_HandlerErrorPropagatesCodeAndDetail guards the
// IPC error contract: when a registered handler returns an error
// that implements ErrorDetailer (and is recognised by status.Code),
// the daemon must classify it AND propagate the detail map through
// the wire so the client sees the same envelope the local path
// would have surfaced.
//
// Inlines the server setup to get a handle on *daemon.Server (the
// shared `withServer` helper only exposes the SubscriptionManager).
func TestServer_HandlerErrorPropagatesCodeAndDetail(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "tgd")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sock := filepath.Join(dir, "d.sock")

	mgr := daemon.NewSubscriptionManager(8)
	srv := daemon.NewServer("alice", sock, mgr, nil)
	require.NoError(t, srv.Listen())

	srv.Register("rate_blow_up", func(_ context.Context, _ json.RawMessage) (json.RawMessage, error) {
		return nil, &session.FloodWaitError{Seconds: 17}
	})

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	go func() {
		_ = srv.Serve(ctx)
		close(doneCh)
	}()
	t.Cleanup(func() {
		cancel()
		_ = srv.Close()
		<-doneCh
		mgr.Close()
	})

	cl := dialClient(t, sock)
	defer func() { _ = cl.Close() }()

	_, err = cl.Call(context.Background(), "rate_blow_up", nil)
	require.Error(t, err)
	var rem *daemon.RemoteError
	require.ErrorAs(t, err, &rem)
	require.Equal(t, "flood_wait", rem.Code, "code must come from status.Code(err), not the generic method_failed")
	require.Equal(t, 6, rem.ExitCode, "exit code must come from status.MapExitCode")
	require.NotNil(t, rem.Detail)
	require.InDelta(t, float64(17), rem.Detail["retry_after_seconds"], 0)
}
