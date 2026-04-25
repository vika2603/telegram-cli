package deletecmd_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	del "github.com/vika2603/telegram-cli/internal/cli/msg/delete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *del.Options
	f := runtime.NewTestInvocation(t)
	cmd := del.New(f, func(o *del.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@a:10", "@a:11", "--revoke", "--yes"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, []string{"@a:10", "@a:11"}, captured.RawMessageRefs)
	require.True(t, captured.Revoke)
	require.True(t, captured.Yes)
}

func TestRun_NoYesNoPrompter_Declined(t *testing.T) {
	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.Prompter = stubPrompter{ok: false}
	opts := &del.Options{
		RawMessageRefs: []string{"@a:1"}, Prompter: f.Prompter, IOStreams: ios,
		Delete: func(_ context.Context, _ actionmessage.DeleteQuery) error { return nil },
	}
	err := del.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestRun_YesSkipsPromptCallsDelete(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	called := false
	opts := &del.Options{
		RawMessageRefs: []string{"@a:1", "@a:2"}, Yes: true, Prompter: f.Prompter, IOStreams: ios,
		Delete: func(_ context.Context, _ actionmessage.DeleteQuery) error { called = true; return nil },
	}
	require.NoError(t, del.Run(context.Background(), opts))
	require.True(t, called)
	require.Contains(t, stdout.String(), "deleted")
}

func TestRun_PromptAcceptedCallsDelete(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.Prompter = stubPrompter{ok: true}
	called := false
	opts := &del.Options{
		RawMessageRefs: []string{"@a:5"}, Prompter: f.Prompter, IOStreams: ios,
		Delete: func(_ context.Context, _ actionmessage.DeleteQuery) error { called = true; return nil },
	}
	require.NoError(t, del.Run(context.Background(), opts))
	require.True(t, called)
	require.Contains(t, stdout.String(), "deleted")
}

func TestRun_NilDeleteClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	opts := &del.Options{RawMessageRefs: []string{"@a:1"}, Yes: true, Prompter: f.Prompter, IOStreams: ios}
	err := del.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_RevokeOutputsRevoked(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	opts := &del.Options{
		RawMessageRefs: []string{"@a:1"}, Revoke: true, Yes: true, Prompter: f.Prompter, IOStreams: ios,
		Delete: func(_ context.Context, _ actionmessage.DeleteQuery) error { return nil },
	}
	require.NoError(t, del.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "revoked")
	require.NotContains(t, stdout.String(), "deleted")
}

type stubPrompter struct{ ok bool }

func (s stubPrompter) Confirm(string, bool) (bool, error)   { return s.ok, nil }
func (s stubPrompter) Password(string) (string, error)      { return "", nil }
func (s stubPrompter) Input(string, string) (string, error) { return "", nil }
