// Package set implements "tg config set <key> <value>".
package set

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
	Key   string
	Value string
	Force bool
	Yes   bool

	F        *runtime.Invocation
	Exporter output.Exporter
}

// New builds the cobra.Command for "tg config set".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Write one config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Key, opts.Value = args[0], args[1]
			if runF != nil {
				return runF(opts)
			}
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Force, "force", false, "Required to modify api_hash")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip destructive confirmation prompt")
	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "key", "old", "new"})
	return cmd
}

// Run writes one config value and emits the result.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionconfig.Set(ctx, actionconfig.SetRequest{
		Key:        opts.Key,
		Value:      opts.Value,
		ConfigPath: opts.F.ConfigPath,
		Force:      opts.Force,
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
