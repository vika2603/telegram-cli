package contact_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/command"
)

type stubPrompter struct{ ok bool }

func (s stubPrompter) Confirm(string, bool) (bool, error)   { return s.ok, nil }
func (s stubPrompter) Password(string) (string, error)      { return "", nil }
func (s stubPrompter) Select(string, []string) (int, error) { return 0, nil }
func (s stubPrompter) Input(string, string) (string, error) { return "", nil }

func TestBlockRequiresFunction(t *testing.T) {
	err := contact.Block(context.Background(), contact.PeerRequest{RawRef: "@bob", Yes: true}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestBlockDeclined(t *testing.T) {
	called := false
	err := contact.Block(context.Background(), contact.PeerRequest{
		RawRef: "@bob", Prompter: stubPrompter{ok: false},
	}, func(context.Context, contact.PeerQuery) error {
		called = true
		return nil
	})
	require.ErrorIs(t, err, command.ErrNotConfirmed)
	require.False(t, called)
}

func TestBlockNormalizesRef(t *testing.T) {
	err := contact.Block(context.Background(), contact.PeerRequest{RawRef: "@bob", Yes: true}, func(_ context.Context, q contact.PeerQuery) error {
		require.Equal(t, "bob", q.Ref.Value)
		return nil
	})
	require.NoError(t, err)
}

func TestDeleteNormalizesRef(t *testing.T) {
	err := contact.Delete(context.Background(), contact.PeerRequest{RawRef: "@alice", Yes: true}, func(_ context.Context, q contact.PeerQuery) error {
		require.Equal(t, "alice", q.Ref.Value)
		return nil
	})
	require.NoError(t, err)
}

func TestUnblockNormalizesRef(t *testing.T) {
	err := contact.Unblock(context.Background(), contact.PeerRequest{RawRef: "@alice"}, func(_ context.Context, q contact.PeerQuery) error {
		require.Equal(t, "alice", q.Ref.Value)
		return nil
	})
	require.NoError(t, err)
}
