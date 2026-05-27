package status_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/daemon/status"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type fakeMgr struct {
	platform string
	status   *daemon.Status
}

func (f *fakeMgr) Platform() string                { return f.platform }
func (f *fakeMgr) Install(daemon.Config) error     { return nil }
func (f *fakeMgr) Uninstall() error                { return nil }
func (f *fakeMgr) Start() error                    { return nil }
func (f *fakeMgr) Stop() error                     { return nil }
func (f *fakeMgr) Restart() error                  { return nil }
func (f *fakeMgr) Status() (*daemon.Status, error) { return f.status, nil }

func TestRun_RejectsEmptyAccount(t *testing.T) {
	ios, _, _, _ := ui.Test()
	err := status.Run(context.Background(), &status.Options{
		IOStreams: ios,
		NewMgr: func(string) (daemon.Manager, error) {
			return &fakeMgr{platform: "fake"}, nil
		},
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_EmitsJSONRow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ios, _, stdout, _ := ui.Test()

	mgr := &fakeMgr{
		platform: "launchd",
		status: &daemon.Status{
			Installed: true, Running: true, PID: 1234,
			Platform: "launchd", Account: "alice",
		},
	}
	err := status.Run(context.Background(), &status.Options{
		Account:   "alice",
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return mgr, nil },
	})
	require.NoError(t, err)

	out := strings.TrimSpace(stdout.String())
	require.NotEmpty(t, out)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.Equal(t, "alice", got["account"])
	require.Equal(t, true, got["installed"])
	require.Equal(t, true, got["running"])
	require.InDelta(t, 1234, got["pid"], 0)
	require.Equal(t, "launchd", got["platform"])
	require.NotEmpty(t, got["updates_file"])
	require.NotEmpty(t, got["socket_path"])
}

func TestRun_NilStatusFromManagerStillEmitsRow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ios, _, stdout, _ := ui.Test()

	mgr := &fakeMgr{platform: "launchd", status: nil}
	err := status.Run(context.Background(), &status.Options{
		Account:   "alice",
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return mgr, nil },
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &got))
	require.Equal(t, "alice", got["account"])
	require.Equal(t, false, got["installed"])
	require.Equal(t, "launchd", got["platform"])
}
