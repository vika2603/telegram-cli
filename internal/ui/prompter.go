package ui

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/term"

	"github.com/vika2603/telegram-cli/internal/command"
)

// Prompter is the interface commands use for interactive input.
// SystemPrompter reads from an IOStreams; StubPrompter returns canned
// answers for tests.
type Prompter interface {
	Password(prompt string) (string, error)
	Confirm(prompt string, defaultAns bool) (bool, error)
	Input(prompt, defaultValue string) (string, error)
}

// SystemPrompter is the production Prompter.
type SystemPrompter struct{ IO *IOStreams }

func (p *SystemPrompter) ensurePromptable() error {
	if p == nil || p.IO == nil {
		return fmt.Errorf("%w: no terminal IO available for prompt", command.ErrPrecondition)
	}
	if !p.IO.CanPrompt() {
		return fmt.Errorf("%w: stdin is not an interactive terminal", command.ErrPrecondition)
	}
	return nil
}

// Password prompts for a secret value.
func (p *SystemPrompter) Password(prompt string) (string, error) {
	if err := p.ensurePromptable(); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(p.IO.ErrOut, "%s: ", prompt); err != nil {
		return "", err
	}
	if f, ok := p.IO.In.(interface{ Fd() uintptr }); ok && p.IO.IsStdinTTY() {
		b, err := term.ReadPassword(int(f.Fd()))
		_, _ = fmt.Fprintln(p.IO.ErrOut)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(b), "\r\n"), nil
	}
	return readPromptLine(p.IO.In)
}

func (p *SystemPrompter) Input(prompt, defaultValue string) (string, error) {
	if err := p.ensurePromptable(); err != nil {
		return "", err
	}
	label := prompt
	if defaultValue != "" {
		label += fmt.Sprintf(" [%s]", defaultValue)
	}
	if _, err := fmt.Fprintf(p.IO.ErrOut, "%s: ", label); err != nil {
		return "", err
	}
	value, err := readPromptLine(p.IO.In)
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func (p *SystemPrompter) Confirm(prompt string, defaultAns bool) (bool, error) {
	if err := p.ensurePromptable(); err != nil {
		return false, err
	}
	suffix := " [y/N]: "
	if defaultAns {
		suffix = " [Y/n]: "
	}
	for {
		if _, err := fmt.Fprint(p.IO.ErrOut, prompt+suffix); err != nil {
			return false, err
		}
		value, err := readPromptLine(p.IO.In)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "":
			return defaultAns, nil
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			if _, err := fmt.Fprintln(p.IO.ErrOut, "please answer yes or no"); err != nil {
				return false, err
			}
		}
	}
}

func readPromptLine(r interface{ Read([]byte) (int, error) }) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
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

var _ Prompter = (*SystemPrompter)(nil)
var _ Prompter = (*StubPrompter)(nil)
