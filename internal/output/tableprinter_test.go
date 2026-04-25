package output_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestTablePrinter_HumanAligned(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	ios.SetStdoutTTY(true)
	tp := output.NewTablePrinter(ios)
	tp.AddHeader("NAME", "STATE", "DEFAULT")
	tp.AddRow("work", "AUTHED", "*")
	tp.AddRow("home-account", "INIT", "")
	require.NoError(t, tp.Render())

	got := stdout.String()
	require.Contains(t, got, "NAME")
	require.Contains(t, got, "work")
	require.Contains(t, got, "home-account")
}

func TestTablePrinter_HumanAlignedWideCharacters(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	ios.SetStdoutTTY(true)
	tp := output.NewTablePrinter(ios)
	tp.AddHeader("NAME", "STATE")
	tp.AddRow("风向旗", "AUTHED")
	tp.AddRow("ascii", "NEW")
	require.NoError(t, tp.Render())

	got := stdout.String()
	require.Contains(t, got, "风向旗")
	require.Contains(t, got, "ascii")
}

func TestTablePrinter_EmptyRendersOnlyHeader(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	ios.SetStdoutTTY(true)
	tp := output.NewTablePrinter(ios)
	tp.AddHeader("A", "B")
	require.NoError(t, tp.Render())
	require.Contains(t, stdout.String(), "A")
	require.Contains(t, stdout.String(), "B")
}

func TestTablePrinter_NonTTYIsTabSeparated(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	// Default IOStreams.Test has stdoutTTY=false.
	tp := output.NewTablePrinter(ios)
	tp.AddHeader("NAME", "STATE")
	tp.AddRow("work", "AUTHED")
	require.NoError(t, tp.Render())
	require.Contains(t, stdout.String(), "NAME\tSTATE")
	require.Contains(t, stdout.String(), "work\tAUTHED")
}
