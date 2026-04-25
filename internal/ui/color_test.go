package ui_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestColorScheme_DisabledIsPassthrough(t *testing.T) {
	ios, _, _, _ := ui.Test() // color disabled by default
	cs := ios.ColorScheme()
	require.Equal(t, "hello", cs.Red("hello"))
	require.Equal(t, "hello", cs.Bold("hello"))
	require.Equal(t, "hello", cs.Green("hello"))
}

func TestColorScheme_EnabledWrapsANSI(t *testing.T) {
	ios, _, _, _ := ui.Test()
	ios.SetColorEnabled(true)
	cs := ios.ColorScheme()
	out := cs.Red("hello")
	require.Contains(t, out, "\x1b[31m")
	require.Contains(t, out, "hello")
	require.Contains(t, out, "\x1b[0m")
}

func TestColorEnabled_FlipsScheme(t *testing.T) {
	ios, _, _, _ := ui.Test()
	require.False(t, ios.ColorEnabled())
	ios.SetColorEnabled(true)
	require.True(t, ios.ColorEnabled())
}
