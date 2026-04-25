// Package show implements the "tg config show" command.
package show

import (
	"fmt"

	"github.com/spf13/cobra"

	actionconfig "github.com/vika2603/telegram-cli/internal/action/config"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// Options holds flag values and injectable dependencies for "config show".
type Options struct {
	Exporter output.Exporter
}

// New constructs the cobra.Command for "config show".
// When runF is nil, production logic (showRun) is used.
// Tests pass a capture lambda to verify flag parsing without touching disk.
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}

	cmd := &cobra.Command{
		Use:          "show",
		Short:        "Print merged config (api_hash redacted)",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return showRun(f, cmd, opts)
		},
	}

	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{
		"version", "default_account", "account_state", "api_id", "api_hash",
		"output", "log", "flood_wait", "aliases",
	})

	return cmd
}

func showRun(f *runtime.Invocation, cmd *cobra.Command, opts *Options) error {
	cfg, err := f.Config()
	if err != nil {
		return err
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if quiet {
		return nil
	}

	if cfg.DefaultAccount != nil && *cfg.DefaultAccount != "" {
		command.WarnLoosePermsByName(f.IOStreams.ErrOut, *cfg.DefaultAccount)
	}
	r, err := actionconfig.Show(cmd.Context(), actionconfig.ShowRequest{Config: cfg})
	if err != nil {
		return err
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(f.IOStreams, r)
	}

	_, _ = fmt.Fprint(f.IOStreams.Out, r.Human())
	return nil
}
