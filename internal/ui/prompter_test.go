package ui_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestStubPrompter_DispensesAnswersInOrder(t *testing.T) {
	p := &ui.StubPrompter{Answers: []any{"alice", true, 2, "bob"}}

	name, err := p.Input("who?", "")
	require.NoError(t, err)
	require.Equal(t, "alice", name)

	ok, err := p.Confirm("sure?", false)
	require.NoError(t, err)
	require.True(t, ok)

	idx, err := p.Select("which?", []string{"a", "b", "c"})
	require.NoError(t, err)
	require.Equal(t, 2, idx)

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

func TestSystemPrompter_SelectAccessible(t *testing.T) {
	ios, stdin, _, _ := ui.Test()
	ios.SetStdinTTY(true)
	stdin.WriteString("2\n")

	got, err := (&ui.SystemPrompter{IO: ios}).Select("Account", []string{"work", "home"})
	require.NoError(t, err)
	require.Equal(t, 1, got)
}

func TestSystemPrompter_RequiresPromptableStdin(t *testing.T) {
	ios, _, _, _ := ui.Test()

	_, err := (&ui.SystemPrompter{IO: ios}).Input("Name", "")
	require.Error(t, err)
}
