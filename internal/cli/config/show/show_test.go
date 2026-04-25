package show_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account/accounttest"
	"github.com/vika2603/telegram-cli/internal/cli/config/show"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// newTestRoot creates a minimal parent command exposing the persistent flags
// that internal/cli/root registers, so subcommands can read them via
// cmd.Flags().GetBool / GetString in tests.
func newTestRoot(sub *cobra.Command) *cobra.Command {
	root := &cobra.Command{Use: "tg", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().Bool("quiet", false, "Suppress stdout")
	root.PersistentFlags().String("config", "", "Config file path")
	root.AddCommand(sub)
	return root
}

// minimalConfig returns a *config.Config with Version set so Human() output
// includes the "version" line.
func minimalConfig() *config.Config {
	v := 1
	cfg := config.Defaults()
	cfg.Version = &v
	return &cfg
}

func TestNew_ParsesFlags(t *testing.T) {
	var captured *show.Options
	f := runtime.NewTestInvocation(t)
	cmd := show.New(f, func(o *show.Options) error {
		captured = o
		return nil
	})
	// --json has NoOptDefVal="*" so must be passed as --json=<field>.
	cmd.SetArgs(strings.Fields("--json=version"))
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.NotNil(t, captured.Exporter)
}

func TestNew_HumanOutput(t *testing.T) {
	accounttest.TempConfigRoot(t)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.Config = func() (*config.Config, error) { return minimalConfig(), nil }

	sub := show.New(f, nil)
	root := newTestRoot(sub)
	root.SetArgs(strings.Fields("show"))
	require.NoError(t, root.Execute())

	out := stdout.String()
	require.Contains(t, out, "version")
}

func TestNew_JSONOutput(t *testing.T) {
	accounttest.TempConfigRoot(t)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.Config = func() (*config.Config, error) { return minimalConfig(), nil }

	sub := show.New(f, nil)
	root := newTestRoot(sub)
	root.SetArgs(strings.Fields("show --json=version"))
	require.NoError(t, root.Execute())

	out := stdout.String()
	require.NotEmpty(t, out)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(out)), &result))
}

func TestNew_QuietSuppresses(t *testing.T) {
	accounttest.TempConfigRoot(t)

	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.Config = func() (*config.Config, error) { return minimalConfig(), nil }

	sub := show.New(f, nil)
	root := newTestRoot(sub)
	root.SetArgs(strings.Fields("show --quiet"))
	require.NoError(t, root.Execute())

	require.Empty(t, stdout.String())
}
