package block_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/cli/contact/block"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type stubPrompter struct{ ok bool }

func (s stubPrompter) Confirm(string, bool) (bool, error)   { return s.ok, nil }
func (s stubPrompter) Password(string) (string, error)      { return "", nil }
func (s stubPrompter) Input(string, string) (string, error) { return "", nil }

func TestNew_FlagParsing(t *testing.T) {
	var captured *block.Options
	f := runtime.NewTestInvocation(t)
	cmd := block.New(f, func(o *block.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@bob", "--yes"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@bob", captured.RawRef)
	require.True(t, captured.Yes)
}

func TestRun_NilBlockClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &block.Options{RawRef: "@bob", Yes: true, IOStreams: ios}
	err := block.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_PromptDeclined(t *testing.T) {
	ios, _, _, _ := ui.Test()
	called := false
	opts := &block.Options{
		RawRef: "@bob", Prompter: stubPrompter{ok: false}, IOStreams: ios,
		Block: func(_ context.Context, _ actioncontact.PeerQuery) error { called = true; return nil },
	}
	err := block.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
	require.False(t, called)
}

func TestRun_YesBypassesPrompt(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &block.Options{
		RawRef: "@bob", Yes: true, IOStreams: ios,
		Block: func(_ context.Context, q actioncontact.PeerQuery) error {
			require.Equal(t, "bob", q.Ref.Value)
			return nil
		},
	}
	require.NoError(t, block.Run(context.Background(), opts))
	require.Equal(t, "blocked\t@bob\n", stdout.String())
}
