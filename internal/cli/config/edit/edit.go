// Package edit implements "tg config edit".
package edit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	actionconfig "github.com/vika2603/telegram-cli/internal/action/config"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// Options holds the resolved dependencies for Run.
type Options struct {
	F *runtime.Invocation

	// ResolveEditor picks the editor command + args. Tests inject a stub that
	// writes predetermined content to the tmp file path passed as the last arg.
	// Production code resolves $VISUAL / $EDITOR / vim / vi.
	ResolveEditor func() ([]string, error)
}

// New builds the cobra.Command for "tg config edit".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open $EDITOR on the config file with rollback on parse failure",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if runF != nil {
				return runF(opts)
			}
			opts.ResolveEditor = actionconfig.DefaultResolveEditor
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{SkipAuthCheck: true})
	return cmd
}

// Run opens an editor through the config action and emits the result.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionconfig.Edit(ctx, actionconfig.EditRequest{
		ConfigPath:    opts.F.ConfigPath,
		IOStreams:     opts.F.IOStreams,
		Prompter:      opts.F.Prompter,
		ResolveEditor: opts.ResolveEditor,
	})
	if err != nil {
		return err
	}
	b, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("marshal output: %w", err)
	}
	_, _ = fmt.Fprintf(opts.F.IOStreams.Out, "%s\n", b)
	return nil
}
