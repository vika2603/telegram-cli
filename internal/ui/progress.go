package ui

import (
	"time"

	"github.com/briandowns/spinner"
)

// StartProgressIndicator shows an ASCII spinner on stderr while work runs.
// No-op when stderr is not a TTY. Safe to call from any goroutine.
func (s *IOStreams) StartProgressIndicator() {
	s.StartProgressIndicatorWithLabel("")
}

// StartProgressIndicatorWithLabel is StartProgressIndicator plus a label
// printed after the spinner glyph.
func (s *IOStreams) StartProgressIndicatorWithLabel(label string) {
	if !s.stderrTTY {
		return
	}
	sp := spinner.New(spinner.CharSets[11], 120*time.Millisecond, spinner.WithWriter(s.ErrOut))
	if label != "" {
		sp.Suffix = " " + label
	}
	sp.Start()
	s.progress = sp
	s.progressEnabled = true
}

// StopProgressIndicator stops the spinner if running.
func (s *IOStreams) StopProgressIndicator() {
	if s.progress == nil {
		return
	}
	s.progress.Stop()
	s.progress = nil
	s.progressEnabled = false
}

// IsProgressIndicatorEnabled reports whether a spinner is active.
func (s *IOStreams) IsProgressIndicatorEnabled() bool { return s.progressEnabled }
