package daemon_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram"
)

func TestSubscriptionManager_PublishFansToAll(t *testing.T) {
	m := daemon.NewSubscriptionManager(4)
	a := m.Subscribe(telegram.WatchFilter{})
	b := m.Subscribe(telegram.WatchFilter{})

	ev := telegram.WatchEvent{Kind: telegram.EventNewMessage, Row: output.MessageRow{ID: 1}}
	m.Publish(ev)

	require.Equal(t, ev, recvOrFail(t, a))
	require.Equal(t, ev, recvOrFail(t, b))
}

func TestSubscriptionManager_FilterByKindAndPeer(t *testing.T) {
	m := daemon.NewSubscriptionManager(4)
	onlyEdits := m.Subscribe(telegram.WatchFilter{
		Kinds: map[telegram.WatchEventKind]struct{}{telegram.EventEditMessage: {}},
	})
	onlyPeer99 := m.Subscribe(telegram.WatchFilter{
		PeerIDs: map[int64]struct{}{-1_000_000_000_099: {}},
	})

	m.Publish(telegram.WatchEvent{Kind: telegram.EventNewMessage, Row: output.MessageRow{FromID: -1_000_000_000_099}})
	m.Publish(telegram.WatchEvent{Kind: telegram.EventEditMessage, Row: output.MessageRow{FromID: 42}})
	m.Publish(telegram.WatchEvent{Kind: telegram.EventEditMessage, Row: output.MessageRow{FromID: -1_000_000_000_099}})

	require.Equal(t, telegram.EventEditMessage, recvOrFail(t, onlyEdits).Kind)
	require.Equal(t, telegram.EventEditMessage, recvOrFail(t, onlyEdits).Kind)
	requireNoEvent(t, onlyEdits)

	require.Equal(t, int64(-1_000_000_000_099), recvOrFail(t, onlyPeer99).Row.FromID)
	require.Equal(t, int64(-1_000_000_000_099), recvOrFail(t, onlyPeer99).Row.FromID)
	requireNoEvent(t, onlyPeer99)
}

func TestSubscriptionManager_UnsubscribeClosesChannel(t *testing.T) {
	m := daemon.NewSubscriptionManager(4)
	sub := m.Subscribe(telegram.WatchFilter{})
	require.True(t, m.Unsubscribe(sub.ID))
	_, open := <-sub.C
	require.False(t, open, "channel should be closed after Unsubscribe")
	require.False(t, m.Unsubscribe(sub.ID), "second Unsubscribe should report not-found")
}

func TestSubscriptionManager_FullBufferDropsAndCountsLag(t *testing.T) {
	m := daemon.NewSubscriptionManager(2)
	sub := m.Subscribe(telegram.WatchFilter{})

	for i := range 5 {
		m.Publish(telegram.WatchEvent{Kind: telegram.EventNewMessage, Row: output.MessageRow{ID: i}})
	}
	// Buffer = 2 ⇒ 3 events were dropped.
	require.Equal(t, uint64(3), sub.Dropped())
	require.Equal(t, uint64(3), sub.ResetDropped())
	require.Equal(t, uint64(0), sub.Dropped())
}

func TestSubscriptionManager_DeleteEventMatchesAnyRow(t *testing.T) {
	m := daemon.NewSubscriptionManager(4)
	sub := m.Subscribe(telegram.WatchFilter{
		PeerIDs: map[int64]struct{}{-1_000_000_000_099: {}},
	})
	m.Publish(telegram.WatchEvent{
		Kind: telegram.EventDeleteMessages,
		Rows: []output.MessageRow{
			{ID: 1, FromID: 42},
			{ID: 2, FromID: -1_000_000_000_099},
		},
	})
	ev := recvOrFail(t, sub)
	require.Equal(t, telegram.EventDeleteMessages, ev.Kind)
	require.Len(t, ev.Rows, 2)
}

func TestSubscriptionManager_LenReflectsLiveCount(t *testing.T) {
	m := daemon.NewSubscriptionManager(2)
	require.Equal(t, 0, m.Len())
	a := m.Subscribe(telegram.WatchFilter{})
	b := m.Subscribe(telegram.WatchFilter{})
	require.Equal(t, 2, m.Len())
	m.Unsubscribe(a.ID)
	require.Equal(t, 1, m.Len())
	m.Unsubscribe(b.ID)
	require.Equal(t, 0, m.Len())
}

func recvOrFail(t *testing.T, sub *daemon.Subscription) telegram.WatchEvent {
	t.Helper()
	select {
	case ev, ok := <-sub.C:
		require.True(t, ok, "channel closed unexpectedly")
		return ev
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for event")
		return telegram.WatchEvent{}
	}
}

func requireNoEvent(t *testing.T, sub *daemon.Subscription) {
	t.Helper()
	select {
	case ev := <-sub.C:
		t.Fatalf("unexpected event: %+v", ev)
	case <-time.After(100 * time.Millisecond):
	}
}
