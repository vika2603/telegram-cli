package install_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/daemon/install"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type fakeMgr struct {
	platform     string
	installCalls []daemon.Config
	installed    bool
}

func (f *fakeMgr) Platform() string { return f.platform }
func (f *fakeMgr) Install(c daemon.Config) error {
	f.installCalls = append(f.installCalls, c)
	f.installed = true
	return nil
}
func (f *fakeMgr) Uninstall() error { return nil }
func (f *fakeMgr) Start() error     { return nil }
func (f *fakeMgr) Stop() error      { return nil }
func (f *fakeMgr) Restart() error   { return nil }
func (f *fakeMgr) Status() (*daemon.Status, error) {
	return &daemon.Status{Installed: f.installed, Platform: f.platform}, nil
}

func TestRun_RejectsEmptyAccount(t *testing.T) {
	ios, _, _, _ := ui.Test()
	err := install.Run(context.Background(), &install.Options{
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return &fakeMgr{platform: "fake"}, nil },
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_InstallsWithDefaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ios, _, _, _ := ui.Test()
	m := &fakeMgr{platform: "fake"}
	err := install.Run(context.Background(), &install.Options{
		Account:   "alice",
		LogMaxMB:  10,
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return m, nil },
	})
	require.NoError(t, err)
	require.Len(t, m.installCalls, 1)
	require.Equal(t, "alice", m.installCalls[0].Account)
	require.NotEmpty(t, m.installCalls[0].BinaryPath, "Resolve should fill BinaryPath")
	require.NotEmpty(t, m.installCalls[0].LogFile)
	require.Equal(t, int64(10*1024*1024), m.installCalls[0].LogMaxSize)

	// Meta sidecar should now be loadable.
	meta, err := daemon.LoadMeta("alice")
	require.NoError(t, err)
	require.Equal(t, "alice", meta.Account)
	require.Equal(t, "fake", meta.Platform)
}

func TestRun_RefusesReinstallWithoutForce(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ios, _, _, _ := ui.Test()
	m := &fakeMgr{platform: "fake", installed: true}
	err := install.Run(context.Background(), &install.Options{
		Account:   "alice",
		LogMaxMB:  10,
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return m, nil },
	})
	require.ErrorIs(t, err, command.ErrPrecondition)
	require.Empty(t, m.installCalls, "Install should not be called when already installed and --force is false")
}

func TestRun_ForceOverridesExistingInstall(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ios, _, _, _ := ui.Test()
	m := &fakeMgr{platform: "fake", installed: true}
	err := install.Run(context.Background(), &install.Options{
		Account:   "alice",
		LogMaxMB:  10,
		Force:     true,
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return m, nil },
	})
	require.NoError(t, err)
	require.Len(t, m.installCalls, 1)
}
