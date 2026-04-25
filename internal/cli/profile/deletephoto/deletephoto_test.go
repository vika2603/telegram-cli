package deletephoto_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/profile/deletephoto"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type stubPrompter struct{ ok bool }

func (s stubPrompter) Confirm(string, bool) (bool, error)   { return s.ok, nil }
func (s stubPrompter) Password(string) (string, error)      { return "", nil }
func (s stubPrompter) Input(string, string) (string, error) { return "", nil }

func TestNew_FlagParsing(t *testing.T) {
	var captured *deletephoto.Options
	f := runtime.NewTestInvocation(t)
	cmd := deletephoto.New(f, func(o *deletephoto.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"--yes"})
	require.NoError(t, cmd.Execute())
	require.True(t, captured.Yes)
}

func TestRun_NilDeleteClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	opts := &deletephoto.Options{Yes: true, F: f, IOStreams: ios}
	err := deletephoto.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_PromptDeclined(t *testing.T) {
	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.Prompter = stubPrompter{ok: false}
	called := false
	opts := &deletephoto.Options{
		F: f, IOStreams: ios,
		Delete: func(_ context.Context) error { called = true; return nil },
	}
	err := deletephoto.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
	require.False(t, called)
}

func TestRun_YesBypassesPrompt(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	opts := &deletephoto.Options{
		Yes: true, F: f, IOStreams: ios,
		Delete: func(_ context.Context) error { return nil },
	}
	require.NoError(t, deletephoto.Run(context.Background(), opts))
	require.Equal(t, "deleted\n", stdout.String())
}

func TestRun_PromptAccepted(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.Prompter = stubPrompter{ok: true}
	opts := &deletephoto.Options{
		F: f, IOStreams: ios,
		Delete: func(_ context.Context) error { return nil },
	}
	require.NoError(t, deletephoto.Run(context.Background(), opts))
	require.Equal(t, "deleted\n", stdout.String())
}
