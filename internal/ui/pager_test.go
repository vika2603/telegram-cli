package ui_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestSetPager_StoresCommand(t *testing.T) {
	ios, _, _, _ := ui.Test()
	ios.SetPager("less -FIRX")
	require.Equal(t, "less -FIRX", ios.PagerCommand())
}

func TestStartPager_NoOpWhenNotTTY(t *testing.T) {
	ios, _, stdout, _ := ui.Test() // stdoutTTY false
	ios.SetPager("nonexistent-binary")
	require.NoError(t, ios.StartPager())
	_, _ = ios.Out.Write([]byte("hello"))
	require.Equal(t, "hello", stdout.String(), "pager bypassed in non-TTY")
	ios.StopPager()
}

func TestStartPager_NoOpWhenCommandEmpty(t *testing.T) {
	ios, _, _, _ := ui.Test()
	ios.SetStdoutTTY(true)
	ios.SetPager("")
	require.NoError(t, ios.StartPager())
	ios.StopPager()
}
