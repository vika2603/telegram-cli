//go:build linux

package daemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

// runSystemctl is a test seam (mirrors the launchd pattern).
var runSystemctl = func(args ...string) (string, error) {
	cmd := exec.Command("systemctl", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// runLoginctl is the seam for the linger check.
var runLoginctl = func(args ...string) (string, error) {
	cmd := exec.Command("loginctl", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

type systemdManager struct {
	account string
}

func newPlatformManager(accountName string) (Manager, error) {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil, errors.New("systemctl not found; systemd is required on Linux (container without systemd: use nohup/tmux instead)")
	}
	return &systemdManager{account: accountName}, nil
}

func (*systemdManager) Platform() string { return "systemd (user)" }

func (m *systemdManager) serviceName() string {
	return "tg-" + m.account + ".service"
}

func (m *systemdManager) unitPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", m.serviceName())
}

func (m *systemdManager) userArgs(args ...string) []string {
	return append([]string{"--user"}, args...)
}

func (m *systemdManager) Install(cfg Config) error {
	if cfg.Account != m.account {
		return fmt.Errorf("manager bound to account %q, refused to install %q", m.account, cfg.Account)
	}
	if err := os.MkdirAll(filepath.Dir(m.unitPath()), 0o755); err != nil {
		return fmt.Errorf("create systemd user dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.LogFile), 0o700); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	if err := os.WriteFile(m.unitPath(), []byte(m.buildUnit(cfg)), 0o644); err != nil {
		return fmt.Errorf("write unit: %w", err)
	}
	for _, args := range [][]string{
		m.userArgs("daemon-reload"),
		m.userArgs("enable", m.serviceName()),
		m.userArgs("restart", m.serviceName()),
	} {
		if out, err := runSystemctl(args...); err != nil {
			return fmt.Errorf("systemctl %s: %s (%w)", strings.Join(args, " "), out, err)
		}
	}
	return nil
}

func (m *systemdManager) Uninstall() error {
	_, _ = runSystemctl(m.userArgs("disable", "--now", m.serviceName())...)
	if err := os.Remove(m.unitPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit: %w", err)
	}
	_, _ = runSystemctl(m.userArgs("daemon-reload")...)
	return nil
}

func (m *systemdManager) Start() error {
	if out, err := runSystemctl(m.userArgs("start", m.serviceName())...); err != nil {
		return fmt.Errorf("start: %s (%w)", out, err)
	}
	return nil
}

func (m *systemdManager) Stop() error {
	if out, err := runSystemctl(m.userArgs("stop", m.serviceName())...); err != nil {
		return fmt.Errorf("stop: %s (%w)", out, err)
	}
	return nil
}

func (m *systemdManager) Restart() error {
	if out, err := runSystemctl(m.userArgs("restart", m.serviceName())...); err != nil {
		return fmt.Errorf("restart: %s (%w)", out, err)
	}
	return nil
}

func (m *systemdManager) Status() (*Status, error) {
	st := &Status{Platform: m.Platform(), Account: m.account}
	if _, err := os.Stat(m.unitPath()); err == nil {
		st.Installed = true
	}

	// systemctl show outputs key=value lines; ActiveState=active and a
	// numeric MainPID are what we need.
	out, _ := runSystemctl(m.userArgs("show", m.serviceName(), "--no-page",
		"--property=ActiveState",
		"--property=MainPID")...)
	for _, line := range strings.Split(out, "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "ActiveState":
			st.Running = strings.TrimSpace(v) == "active"
		case "MainPID":
			if pid, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && pid > 0 {
				st.PID = pid
			}
		}
	}
	return st, nil
}

// buildUnit renders the per-account systemd unit. Environment= entries
// carry PATH and proxy variables; Restart=on-failure keeps the daemon
// alive across transient MTProto errors.
func (m *systemdManager) buildUnit(cfg Config) string {
	var envLines strings.Builder
	if cfg.EnvPATH != "" {
		fmt.Fprintf(&envLines, "Environment=PATH=%s\n", cfg.EnvPATH)
	}
	for k, v := range cfg.EnvExtra {
		fmt.Fprintf(&envLines, "Environment=%s=%s\n", k, v)
	}
	return fmt.Sprintf(`[Unit]
Description=tg daemon (account %s)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s --account %s daemon run
Restart=on-failure
RestartSec=3
StandardOutput=append:%s
StandardError=append:%s
%s
[Install]
WantedBy=default.target
`, cfg.Account, cfg.BinaryPath, cfg.Account, cfg.LogFile, cfg.LogFile, envLines.String())
}

// CheckLinger returns whether linger is enabled for the current user;
// without linger a user-mode systemd unit stops on logout, which is
// almost never what daemon users want. The CLI surfaces a one-line
// warning when this returns false.
func CheckLinger() (enabled bool, username string) {
	u, err := user.Current()
	if err != nil {
		return false, ""
	}
	username = u.Username
	out, err := runLoginctl("show-user", username, "--property=Linger", "--value")
	if err != nil {
		return false, username
	}
	return strings.TrimSpace(out) == "yes", username
}
