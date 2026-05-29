package ui_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestStubPrompter_DispensesAnswersInOrder(t *testing.T) {
	p := &ui.StubPrompter{Answers: []any{"alice", true, "bob"}}

	name, err := p.Input("who?", "")
	require.NoError(t, err)
	require.Equal(t, "alice", name)

	ok, err := p.Confirm("sure?", false)
	require.NoError(t, err)
	require.True(t, ok)

	pw, err := p.Password("pw?")
	require.NoError(t, err)
	require.Equal(t, "bob", pw)
}

func TestStubPrompter_EmptyReturnsError(t *testing.T) {
	p := &ui.StubPrompter{}
	_, err := p.Input("who?", "")
	require.Error(t, err)
}

func TestStubPrompter_TypeMismatchIsError(t *testing.T) {
	p := &ui.StubPrompter{Answers: []any{42}}
	_, err := p.Input("who?", "")
	require.Error(t, err, "int where string expected")
}

func TestSystemPrompter_InputAccessible(t *testing.T) {
	ios, stdin, _, _ := ui.Test()
	ios.SetStdinTTY(true)
	stdin.WriteString("alice\n")

	got, err := (&ui.SystemPrompter{IO: ios}).Input("Name", "")
	require.NoError(t, err)
	require.Equal(t, "alice", got)
}

func TestSystemPrompter_ConfirmAccessible(t *testing.T) {
	ios, stdin, _, _ := ui.Test()
	ios.SetStdinTTY(true)
	stdin.WriteString("y\n")

	got, err := (&ui.SystemPrompter{IO: ios}).Confirm("Continue?", false)
	require.NoError(t, err)
	require.True(t, got)
}

func TestSystemPrompter_RequiresPromptableStdin(t *testing.T) {
	ios, _, _, _ := ui.Test()

	_, err := (&ui.SystemPrompter{IO: ios}).Input("Name", "")
	require.Error(t, err)
}

func TestSystemPrompter_InputCancelledByContext(t *testing.T) {
	// A pipe with no writer blocks the read forever, mimicking a user who
	// started `tg login` and walked away mid-prompt.
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })

	ios, _, _, _ := ui.Test()
	ios.SetStdinTTY(true)
	ios.In = io.NopCloser(pr)

	ctx, cancel := context.WithCancel(context.Background())
	p := &ui.SystemPrompter{IO: ios, Ctx: ctx}

	done := make(chan error, 1)
	go func() {
		_, err := p.Input("Name", "")
		done <- err
	}()

	cancel() // SIGINT-equivalent: the root cancels the prompt's context.
	select {
	case err := <-done:
		require.ErrorIs(t, err, command.ErrCancel)
	case <-time.After(2 * time.Second):
		t.Fatal("Input did not unblock after context cancel")
	}
}
