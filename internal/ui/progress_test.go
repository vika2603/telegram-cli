package ui_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestStartProgressIndicator_NoOpInNonTTY(t *testing.T) {
	ios, _, _, _ := ui.Test()
	ios.StartProgressIndicator()
	require.False(t, ios.IsProgressIndicatorEnabled())
	ios.StopProgressIndicator()
}

func TestStartProgressIndicator_EnabledInTTY(t *testing.T) {
	ios, _, _, _ := ui.Test()
	ios.SetStderrTTY(true) // spinner prints to stderr
	ios.StartProgressIndicator()
	require.True(t, ios.IsProgressIndicatorEnabled())
	ios.StopProgressIndicator()
	require.False(t, ios.IsProgressIndicatorEnabled())
}
