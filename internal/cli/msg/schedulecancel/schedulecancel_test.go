package schedulecancel_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/msg/schedulecancel"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type stubPrompter struct{ ok bool }

func (s stubPrompter) Confirm(string, bool) (bool, error)   { return s.ok, nil }
func (s stubPrompter) Password(string) (string, error)      { return "", nil }
func (s stubPrompter) Select(string, []string) (int, error) { return 0, nil }
func (s stubPrompter) Input(string, string) (string, error) { return "", nil }

func TestNew_FlagParsing(t *testing.T) {
	var captured *schedulecancel.Options
	f := runtime.NewTestInvocation(t)
	cmd := schedulecancel.New(f, func(o *schedulecancel.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@a", "100", "101", "--yes"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@a", captured.RawRef)
	require.Equal(t, []int{100, 101}, captured.IDs)
	require.True(t, captured.Yes)
}

func TestNew_InvalidMessageID(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := schedulecancel.New(f, nil)
	cmd.SetArgs([]string{"@a", "not-a-number", "--yes"})
	err := cmd.Execute()
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_NilCancelClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	opts := &schedulecancel.Options{RawRef: "@a", IDs: []int{1}, Yes: true, Prompter: f.Prompter, IOStreams: ios}
	err := schedulecancel.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_PromptDeclined(t *testing.T) {
	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.Prompter = stubPrompter{ok: false}
	called := false
	opts := &schedulecancel.Options{
		RawRef: "@a", IDs: []int{1}, Prompter: f.Prompter, IOStreams: ios,
		Cancel: func(_ context.Context, _ actionmessage.CancelScheduledQuery) error { called = true; return nil },
	}
	err := schedulecancel.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
	require.False(t, called, "Cancel closure must not run when user declines")
}

func TestRun_PromptAcceptedStubbed(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.Prompter = stubPrompter{ok: true}
	opts := &schedulecancel.Options{
		RawRef: "@a", IDs: []int{1, 2, 3}, Prompter: f.Prompter, IOStreams: ios,
		Cancel: func(_ context.Context, a actionmessage.CancelScheduledQuery) error {
			require.Equal(t, []int{1, 2, 3}, a.IDs)
			require.Equal(t, "a", a.Ref.Value)
			return nil
		},
	}
	require.NoError(t, schedulecancel.Run(context.Background(), opts))
	require.Equal(t, "cancelled\t3\n", stdout.String())
}

func TestRun_YesBypassesPrompt(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t) // no prompter set
	opts := &schedulecancel.Options{
		RawRef: "@a", IDs: []int{7}, Yes: true, Prompter: f.Prompter, IOStreams: ios,
		Cancel: func(_ context.Context, _ actionmessage.CancelScheduledQuery) error { return nil },
	}
	require.NoError(t, schedulecancel.Run(context.Background(), opts))
	require.Equal(t, "cancelled\t1\n", stdout.String())
}
