package daemon_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/telegram"
)

func TestMetrics_NilSafe(t *testing.T) {
	var m *daemon.Metrics
	m.IncUpdates()
	m.SetSubscriptions(7)
	m.RecordRPC("x", time.Millisecond, nil)
	require.Equal(t, daemon.MetricsSnapshot{}, m.Snapshot())
}

func TestMetrics_CountersAndPerMethod(t *testing.T) {
	m := daemon.NewMetrics()

	m.IncUpdates()
	m.IncUpdates()
	m.IncUpdates()

	m.RecordRPC("msg.list", 100*time.Millisecond, nil)
	m.RecordRPC("msg.list", 300*time.Millisecond, nil)
	m.RecordRPC("msg.list", 200*time.Millisecond, errors.New("boom"))
	m.RecordRPC("me.show", 50*time.Millisecond, nil)

	m.SetSubscriptions(2)

	snap := m.Snapshot()
	require.Equal(t, uint64(3), snap.UpdatesReceived)
	require.Equal(t, int64(2), snap.Subscriptions)
	require.GreaterOrEqual(t, snap.UptimeSeconds, 0.0)

	require.Len(t, snap.RPCCalls, 2)
	msgList := snap.RPCCalls["msg.list"]
	require.Equal(t, uint64(3), msgList.Calls)
	require.Equal(t, uint64(1), msgList.Errors)
	require.InDelta(t, 200.0, msgList.AvgLatencyMs, 1.0,
		"avg latency should be (100+300+200)/3 = 200ms")

	meShow := snap.RPCCalls["me.show"]
	require.Equal(t, uint64(1), meShow.Calls)
	require.Equal(t, uint64(0), meShow.Errors)
}

func TestMetrics_SnapshotIsACopy(t *testing.T) {
	m := daemon.NewMetrics()
	m.RecordRPC("a", time.Millisecond, nil)
	first := m.Snapshot()
	m.RecordRPC("a", time.Millisecond, nil)
	require.Equal(t, uint64(1), first.RPCCalls["a"].Calls,
		"prior snapshot must not pick up later writes")
}

// TestDaemonStatsRPC stands up a real server and asks for the stats
// payload via the built-in daemon.stats handler. Exercises the
// dispatch case directly (not the application-registry path).
func TestDaemonStatsRPC(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "tgd-stats")
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

	// Generate two subscriptions so the snapshot's gauge is non-zero.
	sub1 := subs.Subscribe(telegram.WatchFilter{})
	sub2 := subs.Subscribe(telegram.WatchFilter{})
	defer sub1.Close()
	defer sub2.Close()

	cl, err := daemon.Dial(context.Background(), account)
	require.NoError(t, err)
	defer func() { _ = cl.Close() }()

	snap, err := cl.Stats(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, snap.StartedAt)
	require.GreaterOrEqual(t, snap.UptimeSeconds, 0.0)
	require.Equal(t, int64(2), snap.Subscriptions,
		"daemon.stats should reflect live subscription count")

	cancel()
	<-doneCh
}
