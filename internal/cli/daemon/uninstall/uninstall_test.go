package uninstall_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/daemon/uninstall"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type fakeMgr struct {
	uninstallErr error
	calls        int
}

func (f *fakeMgr) Platform() string                { return "fake" }
func (f *fakeMgr) Install(daemon.Config) error     { return nil }
func (f *fakeMgr) Uninstall() error                { f.calls++; return f.uninstallErr }
func (f *fakeMgr) Start() error                    { return nil }
func (f *fakeMgr) Stop() error                     { return nil }
func (f *fakeMgr) Restart() error                  { return nil }
func (f *fakeMgr) Status() (*daemon.Status, error) { return &daemon.Status{}, nil }

func TestRun_RejectsEmptyAccount(t *testing.T) {
	ios, _, _, _ := ui.Test()
	err := uninstall.Run(context.Background(), &uninstall.Options{
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return &fakeMgr{}, nil },
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_RemovesMetaAndCallsManager(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, daemon.SaveMeta(&daemon.Meta{Account: "alice", Platform: "fake"}))

	ios, _, _, _ := ui.Test()
	m := &fakeMgr{}
	err := uninstall.Run(context.Background(), &uninstall.Options{
		Account:   "alice",
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return m, nil },
	})
	require.NoError(t, err)
	require.Equal(t, 1, m.calls)

	_, err = daemon.LoadMeta("alice")
	require.Error(t, err, "meta sidecar should be removed")
}

func TestRun_PropagatesManagerError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ios, _, _, _ := ui.Test()
	want := errors.New("bootout failed")
	m := &fakeMgr{uninstallErr: want}
	err := uninstall.Run(context.Background(), &uninstall.Options{
		Account:   "alice",
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return m, nil },
	})
	require.ErrorIs(t, err, want)
}
