package ui_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestTest_ReturnsBuffers(t *testing.T) {
	ios, stdin, stdout, stderr := ui.Test()
	require.NotNil(t, ios)
	require.NotNil(t, stdin)
	require.NotNil(t, stdout)
	require.NotNil(t, stderr)
	require.False(t, ios.IsStdinTTY())
	require.False(t, ios.IsStdoutTTY())
	require.False(t, ios.IsStderrTTY())
}

func TestSetStdoutTTY_FlipsFlag(t *testing.T) {
	ios, _, _, _ := ui.Test()
	require.False(t, ios.IsStdoutTTY())
	ios.SetStdoutTTY(true)
	require.True(t, ios.IsStdoutTTY())
}

func TestCanPrompt_RequiresTTYAndNotNever(t *testing.T) {
	ios, _, _, _ := ui.Test()
	require.False(t, ios.CanPrompt(), "no stdin TTY => false")

	ios.SetStdinTTY(true)
	require.True(t, ios.CanPrompt(), "stdin TTY, prompt allowed => true")

	ios.SetNeverPrompt(true)
	require.False(t, ios.CanPrompt(), "never-prompt overrides TTY => false")
}

func TestTest_OutWritesGoToStdoutBuffer(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	_, err := ios.Out.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, "hello", stdout.String())
}
