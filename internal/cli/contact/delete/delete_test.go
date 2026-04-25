package deletecmd_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	del "github.com/vika2603/telegram-cli/internal/cli/contact/delete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type stubPrompter struct{ ok bool }

func (s stubPrompter) Confirm(string, bool) (bool, error)   { return s.ok, nil }
func (s stubPrompter) Password(string) (string, error)      { return "", nil }
func (s stubPrompter) Input(string, string) (string, error) { return "", nil }

func TestNew_FlagParsing(t *testing.T) {
	var captured *del.Options
	f := runtime.NewTestInvocation(t)
	cmd := del.New(f, func(o *del.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@alice", "--yes"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@alice", captured.RawRef)
	require.True(t, captured.Yes)
}

func TestRun_NilDeleteClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &del.Options{RawRef: "@alice", Yes: true, IOStreams: ios}
	err := del.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_PromptDeclined(t *testing.T) {
	ios, _, _, _ := ui.Test()
	called := false
	opts := &del.Options{
		RawRef: "@alice", Prompter: stubPrompter{ok: false}, IOStreams: ios,
		Delete: func(_ context.Context, _ actioncontact.PeerQuery) error { called = true; return nil },
	}
	err := del.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
	require.False(t, called, "Delete closure must not run when user declines")
}

func TestRun_PromptAccepted(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &del.Options{
		RawRef: "@alice", Prompter: stubPrompter{ok: true}, IOStreams: ios,
		Delete: func(_ context.Context, q actioncontact.PeerQuery) error {
			require.Equal(t, "alice", q.Ref.Value)
			return nil
		},
	}
	require.NoError(t, del.Run(context.Background(), opts))
	require.Equal(t, "deleted\t@alice\n", stdout.String())
}

func TestRun_YesBypassesPrompt(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &del.Options{
		RawRef: "@alice", Yes: true, IOStreams: ios,
		Delete: func(_ context.Context, _ actioncontact.PeerQuery) error { return nil },
	}
	require.NoError(t, del.Run(context.Background(), opts))
	require.Equal(t, "deleted\t@alice\n", stdout.String())
}
