//go:build darwin

package daemon

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeLaunchctl swaps the package-private runLaunchctl var. Returns
// to the previous value when the closure is called.
func fakeLaunchctl(t *testing.T, handler func(args ...string) (string, error)) {
	t.Helper()
	orig := runLaunchctl
	runLaunchctl = handler
	t.Cleanup(func() { runLaunchctl = orig })
}

func TestLaunchd_LabelIncludesAccount(t *testing.T) {
	m := &launchdManager{account: "alice"}
	require.Equal(t, labelPrefix+".alice", m.label())
}

func TestLaunchd_BuildPlistContainsKeyFields(t *testing.T) {
	m := &launchdManager{account: "alice"}
	cfg := Config{
		Account:    "alice",
		BinaryPath: "/usr/local/bin/tg",
		LogFile:    "/tmp/log",
		EnvPATH:    "/usr/local/bin:/usr/bin",
		EnvExtra:   map[string]string{"HTTPS_PROXY": "http://proxy.example:8080"},
	}
	plist := m.buildPlist(cfg)

	require.Contains(t, plist, "<string>"+m.label()+"</string>")
	require.Contains(t, plist, "<string>/usr/local/bin/tg</string>")
	require.Contains(t, plist, "<string>--account</string>")
	require.Contains(t, plist, "<string>alice</string>")
	require.Contains(t, plist, "<string>daemon</string>")
	require.Contains(t, plist, "<string>run</string>")
	require.Contains(t, plist, "<key>RunAtLoad</key>")
	require.Contains(t, plist, "<key>KeepAlive</key>")
	require.Contains(t, plist, "<string>/tmp/log</string>")
	require.Contains(t, plist, "<key>EnvironmentVariables</key>")
	require.Contains(t, plist, "<key>PATH</key>")
	require.Contains(t, plist, "<string>/usr/local/bin:/usr/bin</string>")
	require.Contains(t, plist, "<key>HTTPS_PROXY</key>")
}

func TestLaunchd_BuildPlistEscapesSpecialChars(t *testing.T) {
	m := &launchdManager{account: "alice"}
	plist := m.buildPlist(Config{
		Account:    "alice",
		BinaryPath: `/path with <ampersand>&"quotes"/tg`,
		LogFile:    "/tmp/log",
	})
	require.Contains(t, plist, "&lt;ampersand&gt;")
	require.Contains(t, plist, "&amp;")
	require.Contains(t, plist, "&#34;quotes&#34;")
	require.NotContains(t, plist, `<ampersand>`)
}

func TestLaunchd_InstallRefusesMismatchedAccount(t *testing.T) {
	m := &launchdManager{account: "alice"}
	err := m.Install(Config{Account: "bob", BinaryPath: "/bin/tg", LogFile: "/tmp/log"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bound to account")
}

func TestLaunchd_StartUsesKickstartTarget(t *testing.T) {
	var called []string
	fakeLaunchctl(t, func(args ...string) (string, error) {
		called = append(called, strings.Join(args, " "))
		return "", nil
	})
	m := &launchdManager{account: "alice"}
	require.NoError(t, m.Start())
	require.Len(t, called, 1)
	require.Contains(t, called[0], "kickstart -kp")
	require.Contains(t, called[0], m.label())
}

func TestLaunchd_StopReturnsErrorWhenLaunchctlFails(t *testing.T) {
	fakeLaunchctl(t, func(args ...string) (string, error) {
		return "service not found", errors.New("exit 1")
	})
	m := &launchdManager{account: "alice"}
	err := m.Stop()
	require.Error(t, err)
	require.Contains(t, err.Error(), "stop:")
}

func TestLaunchd_StatusParsesPIDFromPrint(t *testing.T) {
	fakeLaunchctl(t, func(args ...string) (string, error) {
		if len(args) > 0 && args[0] == "print" {
			return "service = {\n\tactive count = 1\n\tpid = 4321;\n\trefs = 1\n}", nil
		}
		return "", nil
	})
	m := &launchdManager{account: "alice"}
	st, err := m.Status()
	require.NoError(t, err)
	require.Equal(t, 4321, st.PID)
	require.True(t, st.Running)
	require.Equal(t, "alice", st.Account)
}

func TestLaunchd_StatusWithoutPIDIsNotRunning(t *testing.T) {
	fakeLaunchctl(t, func(args ...string) (string, error) {
		return "", errors.New("Could not find service")
	})
	m := &launchdManager{account: "alice"}
	st, err := m.Status()
	require.NoError(t, err)
	require.False(t, st.Running)
	require.Equal(t, 0, st.PID)
}
