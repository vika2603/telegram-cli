//go:build windows

package daemon

import (
	"fmt"
)

// Windows service registration is not yet implemented. The interface
// is filled with stubs that return ErrUnsupported so the CLI can still
// build, and CheckLinger always reports false.
//
// Tracking note: a follow-up should wire schtasks /create /tn or sc.exe
// here. cc-connect's daemon/windows.go is a good reference if/when this
// gets prioritised.

type windowsManager struct {
	account string
}

func newPlatformManager(accountName string) (Manager, error) {
	return &windowsManager{account: accountName}, nil
}

func (*windowsManager) Platform() string { return "windows (unsupported)" }

func (m *windowsManager) Install(Config) error {
	return fmt.Errorf("daemon install on windows is not yet implemented")
}
func (m *windowsManager) Uninstall() error {
	return fmt.Errorf("daemon uninstall on windows is not yet implemented")
}
func (m *windowsManager) Start() error {
	return fmt.Errorf("daemon start on windows is not yet implemented")
}
func (m *windowsManager) Stop() error {
	return fmt.Errorf("daemon stop on windows is not yet implemented")
}
func (m *windowsManager) Restart() error {
	return fmt.Errorf("daemon restart on windows is not yet implemented")
}
func (m *windowsManager) Status() (*Status, error) {
	return &Status{Platform: m.Platform(), Account: m.account}, nil
}

func CheckLinger() (enabled bool, user string) { return false, "" }
