// Package unset implements "tg config unset <key>".
package unset

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	actionconfig "github.com/vika2603/telegram-cli/internal/action/config"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Key string
	Yes bool

	F        *runtime.Invocation
	Exporter output.Exporter
}

// New builds the cobra.Command for "tg config unset".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Remove one config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Key = args[0]
			if runF != nil {
				return runF(opts)
			}
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "key", "old"})
	return cmd
}

// Run removes one config value and emits the result.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionconfig.Unset(ctx, actionconfig.UnsetRequest{
		Key:        opts.Key,
		ConfigPath: opts.F.ConfigPath,
		Yes:        opts.Yes,
		Prompter:   opts.F.Prompter,
	})
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.F.IOStreams, row)
	}
	b, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	_, _ = fmt.Fprintln(opts.F.IOStreams.Out, string(b))
	return nil
}
