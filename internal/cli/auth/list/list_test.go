package list_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/account/accounttest"
	"github.com/vika2603/telegram-cli/internal/cli/auth/list"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_ExecutesHumanOutput(t *testing.T) {
	root := accounttest.TempConfigRoot(t)
	accounttest.SeedAccount(t, root, "work", account.StateAUTHED)

	// ui.Test() returns non-TTY streams, so the table is tab-separated.
	// That is the correct behaviour for piped output; assertions use tab-format.
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios

	cmd := list.New(f, nil)
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())

	got := stdout.String()
	require.Contains(t, got, "NAME")
	require.Contains(t, got, "STATE")
	require.Contains(t, got, "work")
}

func TestNew_JSONFlagActivatesExporter(t *testing.T) {
	var captured *list.Options
	f := runtime.NewTestInvocation(t)
	cmd := list.New(f, func(o *list.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"--json"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.NotNil(t, captured.Exporter)
}
