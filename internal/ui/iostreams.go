// Package ui is the unified terminal I/O abstraction for every command.
// Commands read from and write to IOStreams instead of os.Stdin/Stdout/
// Stderr, which makes TTY detection, color, pagination, prompting, and
// spinners testable. Test() constructs an instance backed by bytes.Buffer
// triples; System() wraps the process's real streams.
package ui

import (
	"bytes"
	"io"
	"os"
)

// IOStreams wraps in/out/err, TTY flags, color capabilities, pager state,
// spinner state, and prompt gating. Commands never touch os.Stdin/Stdout/
// Stderr directly; they call fields/methods on IOStreams.
type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer

	stdinTTY    bool
	stdoutTTY   bool
	stderrTTY   bool
	neverPrompt bool

	colorEnabled   bool
	color256       bool
	colorTruecolor bool
	scheme         *ColorScheme

	pagerCommand string
	pagerProcess *os.Process

	progress        interface{ Stop() }
	progressEnabled bool

	altScreenOn bool
}

// System returns an IOStreams wrapping the real process streams with TTY
// and color capabilities auto-detected.
func System() *IOStreams {
	s := &IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}
	s.stdinTTY = isTerminal(os.Stdin)
	s.stdoutTTY = isTerminal(os.Stdout)
	s.stderrTTY = isTerminal(os.Stderr)
	s.detectColor()
	s.scheme = newColorScheme(s.colorEnabled, s.color256, s.colorTruecolor)
	return s
}

// Test returns an IOStreams backed by three bytes.Buffers. All TTY flags
// default to false. The returned buffers are the same instances referenced
// by In/Out/ErrOut, so the test can inspect them directly.
func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	s := &IOStreams{
		In:     io.NopCloser(in),
		Out:    out,
		ErrOut: errOut,
	}
	s.scheme = newColorScheme(false, false, false)
	return s, in, out, errOut
}

func (s *IOStreams) IsStdinTTY() bool      { return s.stdinTTY }
func (s *IOStreams) IsStdoutTTY() bool     { return s.stdoutTTY }
func (s *IOStreams) IsStderrTTY() bool     { return s.stderrTTY }
func (s *IOStreams) SetStdinTTY(v bool)    { s.stdinTTY = v }
func (s *IOStreams) SetStdoutTTY(v bool)   { s.stdoutTTY = v }
func (s *IOStreams) SetStderrTTY(v bool)   { s.stderrTTY = v }
func (s *IOStreams) CanPrompt() bool       { return s.stdinTTY && !s.neverPrompt }
func (s *IOStreams) SetNeverPrompt(v bool) { s.neverPrompt = v }
