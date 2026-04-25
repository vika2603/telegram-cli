// Package get implements "tg config get <key>".
package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	actionconfig "github.com/vika2603/telegram-cli/internal/action/config"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Key          string
	NoRedact     bool
	ErrorIfUnset bool

	F        *runtime.Invocation
	Exporter output.Exporter
}

// New builds the cobra.Command for "tg config get".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Read one config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Key = args[0]
			if runF != nil {
				return runF(opts)
			}
			return Run(cmd.Context(), opts)
		},
		ValidArgsFunction: func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
			names := make([]string, 0, len(config.Keys))
			for _, k := range config.Keys {
				names = append(names, k.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.Flags().BoolVar(&opts.NoRedact, "no-redact", false, "Print api_hash unmasked (default: masked)")
	cmd.Flags().BoolVar(&opts.ErrorIfUnset, "error-if-unset", false, "Exit 70 if the key has no value set")
	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"key", "value", "source"})
	return cmd
}

// Run gets one config value and emits it.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionconfig.Get(ctx, actionconfig.GetRequest{
		Key:          opts.Key,
		ConfigPath:   opts.F.ConfigPath,
		NoRedact:     opts.NoRedact,
		ErrorIfUnset: opts.ErrorIfUnset,
	})
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.F.IOStreams, row)
	}
	_, _ = fmt.Fprintln(opts.F.IOStreams.Out, actionconfig.HumanValue(row.Value))
	return nil
}
