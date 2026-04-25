package account

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"
	"go.etcd.io/bbolt"
)

func TestOpenPeersDB_busyMapsToErrBusy(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	saved := BoltOpenTimeout
	BoltOpenTimeout = 100 * time.Millisecond
	t.Cleanup(func() { BoltOpenTimeout = saved })

	first, err := OpenPeersDB("alice")
	require.NoError(t, err)
	defer func() { _ = first.Close() }()

	_, err = OpenPeersDB("alice")
	require.ErrorIs(t, err, ErrBusy)
}

func TestOpenPeersDB_initialBucketCreated(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	db, err := OpenPeersDB("alice")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		require.NotNil(t, tx.Bucket([]byte(BucketPeers)))
		return nil
	}))
}

func TestPeerStore_CacheSelf_roundtrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	db, err := OpenPeersDB("alice")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ps := NewPeerStore(db)
	u := &tg.User{
		ID:         42,
		AccessHash: 12345,
		Phone:      "15551234567",
		Username:   "alice",
		FirstName:  "Alice",
	}
	require.NoError(t, ps.CacheSelf(u))

	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketSelf))
		require.NotNil(t, b)
		data := b.Get([]byte("id"))
		require.NotNil(t, data)
		var got tg.User
		require.NoError(t, json.Unmarshal(data, &got))
		require.Equal(t, u.ID, got.ID)
		require.Equal(t, u.AccessHash, got.AccessHash)
		require.Equal(t, u.Phone, got.Phone)
		require.Equal(t, u.Username, got.Username)
		require.Equal(t, u.FirstName, got.FirstName)
		return nil
	}))
}

func TestPeerStore_CacheSelf_overwrite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	db, err := OpenPeersDB("alice")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ps := NewPeerStore(db)
	require.NoError(t, ps.CacheSelf(&tg.User{ID: 1, Username: "old"}))
	require.NoError(t, ps.CacheSelf(&tg.User{ID: 2, Username: "new"}))

	require.NoError(t, db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(BucketSelf))
		data := b.Get([]byte("id"))
		var got tg.User
		require.NoError(t, json.Unmarshal(data, &got))
		require.Equal(t, int64(2), got.ID)
		require.Equal(t, "new", got.Username)
		return nil
	}))
}

func TestPeerStore_CacheSelf_rejectsNil(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	db, err := OpenPeersDB("alice")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ps := NewPeerStore(db)
	err = ps.CacheSelf(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "user is nil")
}

func TestPeerStore_PeersStorageRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	db, err := OpenPeersDB("alice")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ps := NewPeerStore(db)
	key := peers.Key{Prefix: "users_", ID: 42}
	value := peers.Value{AccessHash: 99}
	require.NoError(t, ps.Save(context.Background(), key, value))
	got, found, err := ps.Find(context.Background(), key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, value, got)

	require.NoError(t, ps.SavePhone(context.Background(), "15551234567", key))
	gotKey, gotValue, found, err := ps.FindPhone(context.Background(), "15551234567")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, key, gotKey)
	require.Equal(t, value, gotValue)
}

func TestPeerStore_CacheRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	db, err := OpenPeersDB("alice")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ps := NewPeerStore(db)
	require.NoError(t, ps.SaveUsers(context.Background(), &tg.User{
		ID:         42,
		AccessHash: 99,
		Username:   "alice",
		FirstName:  "Alice",
		Status:     &tg.UserStatusOnline{Expires: 123},
	}))
	user, found, err := ps.FindUser(context.Background(), 42)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "alice", user.Username)
	require.Equal(t, int64(99), user.AccessHash)
	require.IsType(t, &tg.UserStatusOnline{}, user.Status)

	require.NoError(t, ps.SaveChannels(context.Background(), &tg.Channel{
		ID:         100,
		AccessHash: 500,
		Title:      "News",
		Username:   "news",
		Broadcast:  true,
		Photo:      &tg.ChatPhotoEmpty{},
	}))
	channel, found, err := ps.FindChannel(context.Background(), 100)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "News", channel.Title)
	require.True(t, channel.Broadcast)
}

func TestPeerStore_RecentPeersAndMessages(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(AccountDir("alice"), 0700))
	db, err := OpenPeersDB("alice")
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	ps := NewPeerStore(db)
	require.NoError(t, ps.RecordRecentPeer(RecentPeer{Ref: "@alice", Title: "Alice", Username: "alice", Kind: "user"}))
	require.NoError(t, ps.RecordRecentPeer(RecentPeer{Ref: "@news", Title: "News", Username: "news", Kind: "channel"}))
	peers, err := ps.ListRecentPeers("ali", 10)
	require.NoError(t, err)
	require.Len(t, peers, 1)
	require.Equal(t, "@alice", peers[0].Ref)

	require.NoError(t, ps.RecordRecentMessage(RecentMessage{Ref: "@alice:10", PeerRef: "@alice", MessageID: 10, Text: "hello"}))
	require.NoError(t, ps.RecordRecentMessage(RecentMessage{Ref: "@news:20", PeerRef: "@news", MessageID: 20, Text: "daily"}))
	messages, err := ps.ListRecentMessages("daily", 10)
	require.NoError(t, err)
	require.Len(t, messages, 1)
	require.Equal(t, "@news:20", messages[0].Ref)
}
