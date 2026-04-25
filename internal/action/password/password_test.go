package password

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestSetReadsCurrentAndNewPasswordsFromStdin(t *testing.T) {
	ios, in, _, _ := ui.Test()
	_, _ = io.WriteString(in, "old\nnew\n")

	var gotCurrent, gotNext string
	row, err := Set(context.Background(), SetRequest{
		CurrentStdin: true,
		NewStdin:     true,
		IOStreams:    ios,
	}, func(context.Context) (bool, error) {
		return true, nil
	}, func(_ context.Context, current, next, hint string) (bool, error) {
		gotCurrent = current
		gotNext = next
		return true, nil
	})

	require.NoError(t, err)
	require.Equal(t, "old", gotCurrent)
	require.Equal(t, "new", gotNext)
	require.Equal(t, "password_set", row.Action)
	require.True(t, row.HadPrevious)
}

func TestSetReadsPasswordsFromPrompter(t *testing.T) {
	ios, _, _, _ := ui.Test()
	prompter := &ui.StubPrompter{Answers: []any{"old", "new", "new"}}

	var gotCurrent, gotNext string
	row, err := Set(context.Background(), SetRequest{
		IOStreams: ios,
		Prompter:  prompter,
	}, func(context.Context) (bool, error) {
		return true, nil
	}, func(_ context.Context, current, next, hint string) (bool, error) {
		gotCurrent = current
		gotNext = next
		return true, nil
	})

	require.NoError(t, err)
	require.Equal(t, "old", gotCurrent)
	require.Equal(t, "new", gotNext)
	require.Equal(t, "password_set", row.Action)
}

func TestSetPrompterPasswordMismatch(t *testing.T) {
	ios, _, _, _ := ui.Test()
	prompter := &ui.StubPrompter{Answers: []any{"new", "again"}}

	_, err := Set(context.Background(), SetRequest{
		IOStreams: ios,
		Prompter:  prompter,
	}, func(context.Context) (bool, error) {
		return false, nil
	}, func(context.Context, string, string, string) (bool, error) {
		t.Fatal("apply must not run")
		return false, nil
	})

	require.ErrorIs(t, err, command.ErrUsage)
}

func TestDisableReadsCurrentPasswordFromPrompter(t *testing.T) {
	ios, _, _, _ := ui.Test()
	prompter := &ui.StubPrompter{Answers: []any{"old", true}}

	var gotCurrent string
	row, err := Disable(context.Background(), DisableRequest{
		IOStreams: ios,
		Prompter:  prompter,
	}, func(context.Context) (bool, error) {
		return true, nil
	}, func(_ context.Context, current string) error {
		gotCurrent = current
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, "old", gotCurrent)
	require.Equal(t, "password_disable", row.Action)
}

func TestDisableStopsWhenNoPassword(t *testing.T) {
	ios, _, _, _ := ui.Test()

	_, err := Disable(context.Background(), DisableRequest{
		CurrentStdin: true,
		Yes:          true,
		IOStreams:    ios,
		Prompter:     &ui.StubPrompter{},
	}, func(context.Context) (bool, error) {
		return false, nil
	}, func(context.Context, string) error {
		t.Fatal("apply must not run")
		return nil
	})

	require.ErrorIs(t, err, command.ErrPrecondition)
}
