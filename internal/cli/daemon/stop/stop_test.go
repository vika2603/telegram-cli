package stop_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/daemon/stop"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/daemon"
	"github.com/vika2603/telegram-cli/internal/ui"
)

type fakeMgr struct {
	stopErr error
	stops   int
}

func (f *fakeMgr) Platform() string                { return "fake" }
func (f *fakeMgr) Install(daemon.Config) error     { return nil }
func (f *fakeMgr) Uninstall() error                { return nil }
func (f *fakeMgr) Start() error                    { return nil }
func (f *fakeMgr) Stop() error                     { f.stops++; return f.stopErr }
func (f *fakeMgr) Restart() error                  { return nil }
func (f *fakeMgr) Status() (*daemon.Status, error) { return &daemon.Status{}, nil }

func TestRun_RejectsEmptyAccount(t *testing.T) {
	ios, _, _, _ := ui.Test()
	err := stop.Run(context.Background(), &stop.Options{
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return &fakeMgr{}, nil },
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_DelegatesToManager(t *testing.T) {
	ios, _, _, _ := ui.Test()
	m := &fakeMgr{}
	err := stop.Run(context.Background(), &stop.Options{
		Account:   "alice",
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return m, nil },
	})
	require.NoError(t, err)
	require.Equal(t, 1, m.stops)
}

func TestRun_PropagatesError(t *testing.T) {
	ios, _, _, _ := ui.Test()
	want := errors.New("bootout failed")
	m := &fakeMgr{stopErr: want}
	err := stop.Run(context.Background(), &stop.Options{
		Account:   "alice",
		IOStreams: ios,
		NewMgr:    func(string) (daemon.Manager, error) { return m, nil },
	})
	require.ErrorIs(t, err, want)
}
