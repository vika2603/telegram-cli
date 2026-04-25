package ui

import (
	"os"

	"golang.org/x/term"
)

// isTerminal probes whether the given file is attached to a terminal.
// Used by System() to populate TTY flags.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// TerminalWidth returns the terminal's column count, or a default of 80
// when stdout is not a terminal or the platform call fails.
func (s *IOStreams) TerminalWidth() int {
	if !s.stdoutTTY {
		return 80
	}
	// Safe: stdoutTTY is only true when Out was derived from an *os.File,
	// which System() enforces. Test() never sets stdoutTTY=true.
	fd := int(os.Stdout.Fd())
	w, _, err := term.GetSize(fd)
	if err != nil || w == 0 {
		return 80
	}
	return w
}

// StartAlternateScreenBuffer switches the terminal into xterm alternate
// screen mode when stdout is a TTY. No-op otherwise. Must be paired with
// StopAlternateScreenBuffer.
func (s *IOStreams) StartAlternateScreenBuffer() {
	if !s.stdoutTTY {
		return
	}
	_, _ = s.Out.Write([]byte("\x1b[?1049h"))
	s.altScreenOn = true
}

// StopAlternateScreenBuffer restores the primary screen buffer.
func (s *IOStreams) StopAlternateScreenBuffer() {
	if !s.altScreenOn {
		return
	}
	_, _ = s.Out.Write([]byte("\x1b[?1049l"))
	s.altScreenOn = false
}

// RefreshScreen clears the terminal for redraw. No-op in non-TTY.
func (s *IOStreams) RefreshScreen() {
	if !s.stdoutTTY {
		return
	}
	_, _ = s.Out.Write([]byte("\x1b[H\x1b[2J"))
}
