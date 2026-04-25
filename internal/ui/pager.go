package ui

import (
	"io"
	"os"
	"os/exec"
)

// SetPager stores the command used by StartPager. Empty disables.
func (s *IOStreams) SetPager(cmd string) { s.pagerCommand = cmd }

// PagerCommand returns the currently configured pager command.
func (s *IOStreams) PagerCommand() string { return s.pagerCommand }

// StartPager launches the configured pager and redirects Out through its
// stdin. No-op when stdout is not a TTY or pagerCommand is empty.
func (s *IOStreams) StartPager() error {
	if !s.stdoutTTY || s.pagerCommand == "" {
		return nil
	}
	cmd := exec.Command("sh", "-c", s.pagerCommand)
	pr, pw := io.Pipe()
	cmd.Stdin = pr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return err
	}
	s.pagerProcess = cmd.Process
	s.Out = pw
	return nil
}

// StopPager closes the pager pipe and waits for the pager process to exit.
// No-op when the pager is not running.
func (s *IOStreams) StopPager() {
	if s.pagerProcess == nil {
		return
	}
	if closer, ok := s.Out.(io.Closer); ok {
		_ = closer.Close()
	}
	_, _ = s.pagerProcess.Wait()
	s.pagerProcess = nil
	s.Out = os.Stdout
}
