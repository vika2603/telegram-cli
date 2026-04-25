package session_test

import (
	"context"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

func TestFakeClient_SelfReturnsConfiguredUser(t *testing.T) {
	fc := &session.FakeClient{SelfValue: tg.User{ID: 42, Username: "alice"}}
	require.Equal(t, int64(42), fc.Self().ID)
	require.Equal(t, "alice", fc.Self().Username)
}

func TestFakeClient_DefaultsReturnUnsupported(t *testing.T) {
	fc := &session.FakeClient{}
	_, err := fc.ResolvePeer(context.Background(), ref.Ref{})
	require.ErrorIs(t, err, command.ErrUnsupported)
}

func TestFakeClient_ResolvePeerUsesOverride(t *testing.T) {
	called := false
	fc := &session.FakeClient{
		ResolvePeerFn: func(_ context.Context, r ref.Ref) (tg.InputPeerClass, error) {
			called = true
			return &tg.InputPeerSelf{}, nil
		},
	}
	_, err := fc.ResolvePeer(context.Background(), ref.Ref{})
	require.NoError(t, err)
	require.True(t, called)
}
