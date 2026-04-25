package peer_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	gotdtgerr "github.com/gotd/td/tgerr"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

func TestNew_NilManagerIsError(t *testing.T) {
	_, err := peer.New(nil, nil, 0)
	require.Error(t, err)
}

func TestResolve_SelfReturnsSelfID(t *testing.T) {
	mgr := peers.Options{}.Build(tg.NewClient(stubInvoker{}))
	r, err := peer.New(mgr, nil, 123)
	require.NoError(t, err)

	got, err := r.Resolve(context.Background(), ref.Ref{Kind: ref.RefKindMe})
	require.NoError(t, err)
	require.Equal(t, int64(123), got.ID)
	require.Equal(t, "user", got.Kind)
}

func TestResolve_InvalidKindIsUsage(t *testing.T) {
	mgr := peers.Options{}.Build(tg.NewClient(stubInvoker{}))
	r, err := peer.New(mgr, nil, 1)
	require.NoError(t, err)

	_, err = r.Resolve(context.Background(), ref.Ref{Kind: ref.RefKindInvalid})
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestMapResolveErr_mapsRPCUsernameErrorsToPeerNotFound(t *testing.T) {
	// The server reports USERNAME_NOT_OCCUPIED / USERNAME_INVALID /
	// CHANNEL_PRIVATE as RPC errors, not as gotd's PeerNotFoundError
	// struct. Our CLI should still map them to ErrPeerNotFound so exit
	// code 66 and the "peer_not_found" JSON code are consistent with
	// what the user sees for structurally-unresolvable refs.
	cases := []string{
		"USERNAME_NOT_OCCUPIED",
		"USERNAME_INVALID",
		"CHANNEL_PRIVATE",
	}
	for _, typ := range cases {
		t.Run(typ, func(t *testing.T) {
			rpc := gotdtgerr.New(400, typ)
			wrapped := fmt.Errorf("resolve: %w", rpc)
			got := peer.MapResolveErrForTest(wrapped, "@whatever")
			require.Error(t, got)
			require.ErrorIs(t, got, peer.ErrNotFound, "type=%s must map to ErrPeerNotFound", typ)
		})
	}
}

func TestMapResolveErr_passesThroughUnrelatedErrors(t *testing.T) {
	sentinel := errors.New("transient boom")
	got := peer.MapResolveErrForTest(sentinel, "@whatever")
	require.Same(t, sentinel, got, "non-peer errors must be passed through unchanged")
}

func TestNormalizeInputPeerID_matchesChatListConvention(t *testing.T) {
	// Resolved.ID must match the signed-int64 convention emitted by
	// `tg chat list` (users positive; chats -chatID; channels
	// -1e12 - channelID) so downstream scripts can cross-reference
	// `chat show` and `chat list` rows by ID.
	cases := []struct {
		name string
		in   tg.InputPeerClass
		want int64
	}{
		{"user", &tg.InputPeerUser{UserID: 42}, 42},
		{"chat", &tg.InputPeerChat{ChatID: 77}, -77},
		{"channel durov", &tg.InputPeerChannel{ChannelID: 1006503122}, -1001006503122},
		{"self", &tg.InputPeerSelf{}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, peer.NormalizeInputPeerIDForTest(tc.in))
		})
	}
}

func TestResolve_IDCacheMissReturnsCacheMiss(t *testing.T) {
	mgr := peers.Options{}.Build(tg.NewClient(stubInvoker{}))
	r, err := peer.New(mgr, nil, 1)
	require.NoError(t, err)

	_, err = r.Resolve(context.Background(), ref.Ref{Kind: ref.RefKindID, ID: 42})
	require.Error(t, err)
	require.ErrorIs(t, err, peer.ErrCacheMiss)
}

func TestResolve_IDFallsBackToGotdPeerManager(t *testing.T) {
	store := &peers.InmemoryStorage{}
	cache := &peers.InmemoryCache{}
	mgr := peers.Options{Storage: store, Cache: cache}.Build(tg.NewClient(stubInvoker{}))
	require.NoError(t, mgr.Apply(context.Background(), []tg.UserClass{
		&tg.User{ID: 42, AccessHash: 9001, FirstName: "Ada", Username: "ada"},
	}, nil))
	r, err := peer.New(mgr, account.NewPeerStore(nil), 1)
	require.NoError(t, err)

	got, err := r.Resolve(context.Background(), ref.Ref{Kind: ref.RefKindID, ID: 42})
	require.NoError(t, err)
	require.Equal(t, int64(42), got.ID)
	require.Equal(t, "user", got.Kind)
	require.Equal(t, "ada", got.Username)
	require.IsType(t, &tg.InputPeerUser{}, got.InputPeer)
}

func TestResolve_DirectPeerRefs(t *testing.T) {
	mgr := peers.Options{}.Build(tg.NewClient(stubInvoker{}))
	r, err := peer.New(mgr, nil, 1)
	require.NoError(t, err)

	user, err := r.Resolve(context.Background(), ref.Ref{Kind: ref.RefKindPeer, Value: "user", ID: 42, AccessHash: 9001})
	require.NoError(t, err)
	require.Equal(t, int64(42), user.ID)
	require.IsType(t, &tg.InputPeerUser{}, user.InputPeer)

	chat, err := r.Resolve(context.Background(), ref.Ref{Kind: ref.RefKindPeer, Value: "chat", ID: 77})
	require.NoError(t, err)
	require.Equal(t, int64(-77), chat.ID)
	require.IsType(t, &tg.InputPeerChat{}, chat.InputPeer)

	channel, err := r.Resolve(context.Background(), ref.Ref{Kind: ref.RefKindPeer, Value: "channel", ID: 100, AccessHash: 555})
	require.NoError(t, err)
	require.Equal(t, int64(-1_000_000_000_100), channel.ID)
	require.IsType(t, &tg.InputPeerChannel{}, channel.InputPeer)
}
