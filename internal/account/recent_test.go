package account

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecentStore_PeersAndMessagesRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, AddAccount(Meta{Name: "work", State: StateNEW}))
	store, err := OpenRecentStore("work")
	require.NoError(t, err)

	require.NoError(t, store.RecordRecentPeer(RecentPeer{
		Ref: "@alice", Title: "Alice", Username: "alice", Kind: "user",
	}))
	require.NoError(t, store.RecordRecentPeer(RecentPeer{
		Ref: "@news", Title: "News", Username: "news", Kind: "channel",
	}))
	peers, err := store.ListRecentPeers("ali", 10)
	require.NoError(t, err)
	require.Len(t, peers, 1)
	require.Equal(t, "@alice", peers[0].Ref)

	require.NoError(t, store.RecordRecentMessage(RecentMessage{
		Ref: "@alice:10", PeerRef: "@alice", MessageID: 10, Text: "hello",
	}))
	require.NoError(t, store.RecordRecentMessage(RecentMessage{
		Ref: "@news:20", PeerRef: "@news", MessageID: 20, Text: "daily",
	}))
	messages, err := store.ListRecentMessages("daily", 10)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, "@news:20", messages[0].Ref)
}

func TestOpenRecentStoreReadOnly_Missing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, err := OpenRecentStoreReadOnly("missing")
	require.ErrorIs(t, err, os.ErrNotExist)
}
