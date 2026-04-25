package ui_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestConfirmDestructive_YesSkipsPrompt(t *testing.T) {
	called := false
	p := &stubPrompter{confirm: func(string, bool) (bool, error) {
		called = true
		return true, nil
	}}
	err := ui.ConfirmDestructive(p, "delete 1 message?", true)
	require.NoError(t, err)
	require.False(t, called, "Prompter must not be called when --yes is set")
}

func TestConfirmDestructive_AcceptedReturnsNil(t *testing.T) {
	p := &stubPrompter{confirm: func(string, bool) (bool, error) {
		return true, nil
	}}
	require.NoError(t, ui.ConfirmDestructive(p, "delete 1 message?", false))
}

func TestConfirmDestructive_DeclinedReturnsErrNotConfirmed(t *testing.T) {
	p := &stubPrompter{confirm: func(string, bool) (bool, error) {
		return false, nil
	}}
	err := ui.ConfirmDestructive(p, "delete 1 message?", false)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestConfirmDestructive_PrompterErrorPropagates(t *testing.T) {
	sentinel := errors.New("boom")
	p := &stubPrompter{confirm: func(string, bool) (bool, error) {
		return false, sentinel
	}}
	err := ui.ConfirmDestructive(p, "delete 1 message?", false)
	require.ErrorIs(t, err, sentinel)
}

type stubPrompter struct {
	confirm func(string, bool) (bool, error)
}

func (s *stubPrompter) Confirm(prompt string, def bool) (bool, error) {
	return s.confirm(prompt, def)
}
func (s *stubPrompter) Password(string) (string, error)      { return "", nil }
func (s *stubPrompter) Select(string, []string) (int, error) { return 0, nil }
func (s *stubPrompter) Input(string, string) (string, error) { return "", nil }
