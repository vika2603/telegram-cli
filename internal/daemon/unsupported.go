//go:build !darwin && !linux && !windows

package daemon

import "fmt"

// Catch-all for unsupported OSes (FreeBSD, OpenBSD, etc.). Returning a
// stub Manager lets the CLI still print a usable error from Install
// rather than failing to compile.

type unsupportedManager struct {
	account string
}

func newPlatformManager(accountName string) (Manager, error) {
	return &unsupportedManager{account: accountName}, nil
}

func (*unsupportedManager) Platform() string { return "unsupported" }

func (*unsupportedManager) Install(Config) error { return errUnsupported() }
func (*unsupportedManager) Uninstall() error     { return errUnsupported() }
func (*unsupportedManager) Start() error         { return errUnsupported() }
func (*unsupportedManager) Stop() error          { return errUnsupported() }
func (*unsupportedManager) Restart() error       { return errUnsupported() }
func (m *unsupportedManager) Status() (*Status, error) {
	return &Status{Platform: m.Platform(), Account: m.account}, nil
}

func errUnsupported() error {
	return fmt.Errorf("daemon management is not supported on this OS")
}

func CheckLinger() (enabled bool, user string) { return false, "" }
