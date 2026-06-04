//go:build darwin

package daemon

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// runLaunchctl is a var so tests can stub it without spawning real
// processes. Production wiring shells out to /bin/launchctl.
var runLaunchctl = func(args ...string) (string, error) {
	cmd := exec.Command("launchctl", args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// labelPrefix is the launchd reverse-DNS prefix. The actual label
// includes the account: e.g. "com.vika2603.telegram-cli.<account>".
const labelPrefix = "com.vika2603.telegram-cli"

type launchdManager struct {
	account string
}

func newPlatformManager(accountName string) (Manager, error) {
	return &launchdManager{account: accountName}, nil
}

func (*launchdManager) Platform() string { return "launchd" }

func (m *launchdManager) label() string {
	return labelPrefix + "." + m.account
}

func (m *launchdManager) plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", m.label()+".plist")
}

func (m *launchdManager) Install(cfg Config) error {
	if cfg.Account != m.account {
		return fmt.Errorf("manager bound to account %q, refused to install %q", m.account, cfg.Account)
	}
	plistPath := m.plistPath()
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(cfg.LogFile), 0o700); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	// Unload any pre-existing job so install is idempotent and we never
	// leave a stale plist active under a different binary path.
	_, _ = runLaunchctl("bootout", m.guiTarget())

	plist := m.buildPlist(cfg)
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	if out, err := runLaunchctl("bootstrap", m.guiDomain(), plistPath); err != nil {
		return fmt.Errorf("launchctl bootstrap: %s (%w)", out, err)
	}
	if out, err := runLaunchctl("kickstart", "-kp", m.guiTarget()); err != nil {
		return fmt.Errorf("launchctl kickstart: %s (%w)", out, err)
	}
	return nil
}

func (m *launchdManager) Uninstall() error {
	_, _ = runLaunchctl("bootout", m.guiTarget())
	plistPath := m.plistPath()
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}

func (m *launchdManager) Start() error {
	if out, err := runLaunchctl("kickstart", "-kp", m.guiTarget()); err != nil {
		// Fallback: bootstrap if not loaded.
		if _, err2 := runLaunchctl("bootstrap", m.guiDomain(), m.plistPath()); err2 != nil {
			return fmt.Errorf("start: %s (%w)", out, err)
		}
		return nil
	}
	return nil
}

func (m *launchdManager) Stop() error {
	if out, err := runLaunchctl("bootout", m.guiTarget()); err != nil {
		return fmt.Errorf("stop: %s (%w)", out, err)
	}
	return nil
}

func (m *launchdManager) Restart() error {
	if out, err := runLaunchctl("kickstart", "-kp", m.guiTarget()); err != nil {
		return fmt.Errorf("restart: %s (%w)", out, err)
	}
	return nil
}

func (m *launchdManager) Status() (*Status, error) {
	st := &Status{Platform: m.Platform(), Account: m.account}
	if _, err := os.Stat(m.plistPath()); err == nil {
		st.Installed = true
	}
	// "launchctl print" failure is normal-but-informational: the agent
	// is not bootstrapped, so Running stays false. The plist-on-disk
	// check above already determined Installed.
	out, runErr := runLaunchctl("print", m.guiTarget())
	if runErr != nil {
		return st, nil //nolint:nilerr // not-loaded is a state, not an error.
	}
	// launchctl print emits a key/value tree; parse out "pid =" if
	// present. Anything else we surface via the status string would be
	// noise.
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "pid =") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if pid, perr := strconv.Atoi(strings.TrimSuffix(parts[2], ";")); perr == nil {
					st.PID = pid
					st.Running = pid > 0
				}
			}
		}
	}
	return st, nil
}

// guiDomain returns the per-user launchd domain (gui/<uid>) appropriate
// for an Agent. System-level Daemons would target "system/" instead;
// tg's per-user account model fits the GUI/agent path.
func (m *launchdManager) guiDomain() string {
	return "gui/" + strconv.Itoa(os.Getuid())
}

func (m *launchdManager) guiTarget() string {
	return m.guiDomain() + "/" + m.label()
}

// buildPlist renders the per-account LaunchAgent definition. The
// resulting XML is small enough to construct by hand — we go through
// encoding/xml for the key/value array values to avoid mishandling
// embedded quotes in PATH or proxy env values.
func (m *launchdManager) buildPlist(cfg Config) string {
	env := map[string]string{}
	if cfg.EnvPATH != "" {
		env["PATH"] = cfg.EnvPATH
	}
	for k, v := range cfg.EnvExtra {
		env[k] = v
	}

	var envXML strings.Builder
	if len(env) > 0 {
		envXML.WriteString("    <key>EnvironmentVariables</key>\n    <dict>\n")
		for k, v := range env {
			kE, _ := xmlEscape(k)
			vE, _ := xmlEscape(v)
			fmt.Fprintf(&envXML, "      <key>%s</key>\n      <string>%s</string>\n", kE, vE)
		}
		envXML.WriteString("    </dict>\n")
	}

	binE, _ := xmlEscape(cfg.BinaryPath)
	accE, _ := xmlEscape(cfg.Account)
	logE, _ := xmlEscape(cfg.LogFile)
	labelE, _ := xmlEscape(m.label())

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
      <string>%s</string>
      <string>--account</string>
      <string>%s</string>
      <string>daemon</string>
      <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>ProcessType</key>
    <string>Background</string>
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
%s  </dict>
</plist>
`, labelE, binE, accE, logE, logE, envXML.String())
}

// xmlEscape wraps encoding/xml's Escape with the simpler returnable
// signature we use in plist construction. The error is the writer
// error, which a strings.Builder never produces; we still keep the
// return for future-proofing.
func xmlEscape(s string) (string, error) {
	var buf strings.Builder
	if err := xml.EscapeText(&buf, []byte(s)); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CheckLinger is a no-op on darwin (Mac has no equivalent concept).
// Present so the CLI can call it uniformly across platforms.
func CheckLinger() (enabled bool, user string) { return false, "" }
