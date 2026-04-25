package ui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/huh/v2"

	"github.com/vika2603/telegram-cli/internal/command"
)

// Prompter is the interface commands use for interactive input.
// SystemPrompter reads from an IOStreams; StubPrompter returns canned
// answers for tests.
type Prompter interface {
	Password(prompt string) (string, error)
	Confirm(prompt string, defaultAns bool) (bool, error)
	Select(prompt string, options []string) (int, error)
	Input(prompt, defaultValue string) (string, error)
}

// SystemPrompter is the production Prompter. It delegates prompt rendering,
// selection, validation, terminal key handling, and password echo behavior to
// Charm's huh package while keeping the rest of the codebase behind the small
// Prompter interface.
type SystemPrompter struct{ IO *IOStreams }

func (p *SystemPrompter) run(field huh.Field) error {
	if p == nil || p.IO == nil {
		return fmt.Errorf("%w: no terminal IO available for prompt", command.ErrPrecondition)
	}
	if !p.IO.CanPrompt() {
		return fmt.Errorf("%w: stdin is not an interactive terminal", command.ErrPrecondition)
	}
	form := huh.NewForm(huh.NewGroup(field)).
		WithInput(p.IO.In).
		WithOutput(p.IO.ErrOut).
		WithShowHelp(false).
		WithWidth(p.IO.TerminalWidth())
	if !p.IO.IsStderrTTY() {
		form.WithAccessible(true)
	}
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return fmt.Errorf("%w", command.ErrCancel)
		}
		return err
	}
	return nil
}

// Password prompts for a secret value.
func (p *SystemPrompter) Password(prompt string) (string, error) {
	var value string
	field := huh.NewInput().
		Title(prompt).
		Value(&value).
		EchoMode(huh.EchoModeNone).
		Inline(true)
	if err := p.run(field); err != nil {
		return "", err
	}
	return value, nil
}

func (p *SystemPrompter) Confirm(prompt string, defaultAns bool) (bool, error) {
	value := defaultAns
	field := huh.NewConfirm().
		Title(prompt).
		Value(&value).
		Affirmative("Yes").
		Negative("No").
		Inline(true)
	if err := p.run(field); err != nil {
		return false, err
	}
	return value, nil
}

func (p *SystemPrompter) Select(prompt string, options []string) (int, error) {
	if len(options) == 0 {
		return -1, errors.New("selection requires at least one option")
	}
	value := 0
	opts := make([]huh.Option[int], 0, len(options))
	for i, option := range options {
		opts = append(opts, huh.NewOption(option, i))
	}
	height := len(options)
	if height > 10 {
		height = 10
	}
	field := huh.NewSelect[int]().
		Title(prompt).
		Options(opts...).
		Value(&value).
		Height(height)
	if err := p.run(field); err != nil {
		return -1, err
	}
	return value, nil
}

func (p *SystemPrompter) Input(prompt, defaultValue string) (string, error) {
	value := defaultValue
	field := huh.NewInput().
		Title(prompt).
		Value(&value).
		Inline(true)
	if err := p.run(field); err != nil {
		return "", err
	}
	return strings.TrimRight(value, "\r\n"), nil
}

// StubPrompter dispenses canned answers in order. Every method pops the
// first Answers element; returns error if empty or type mismatch.
type StubPrompter struct {
	Answers []any
}

func (p *StubPrompter) next() (any, error) {
	if len(p.Answers) == 0 {
		return nil, errors.New("StubPrompter: no more answers")
	}
	v := p.Answers[0]
	p.Answers = p.Answers[1:]
	return v, nil
}

func (p *StubPrompter) Input(_, _ string) (string, error) {
	v, err := p.next()
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("StubPrompter.Input: expected string, got %T", v)
	}
	return s, nil
}

func (p *StubPrompter) Password(_ string) (string, error) { return p.Input("", "") }

func (p *StubPrompter) Confirm(_ string, _ bool) (bool, error) {
	v, err := p.next()
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("StubPrompter.Confirm: expected bool, got %T", v)
	}
	return b, nil
}

func (p *StubPrompter) Select(_ string, _ []string) (int, error) {
	v, err := p.next()
	if err != nil {
		return -1, err
	}
	i, ok := v.(int)
	if !ok {
		return -1, fmt.Errorf("StubPrompter.Select: expected int, got %T", v)
	}
	return i, nil
}

var _ Prompter = (*SystemPrompter)(nil)
var _ Prompter = (*StubPrompter)(nil)
