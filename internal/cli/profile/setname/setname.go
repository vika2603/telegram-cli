// Package setname implements "tg profile set-name <first>".
package setname

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionprofile "github.com/vika2603/telegram-cli/internal/action/profile"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	First   string
	Last    string
	LastSet bool // true when --last was explicitly passed (including "")

	IOStreams *ui.IOStreams
	Exporter  output.Exporter

	// Update is the closure that performs the actual Telegram call. Production
	// code sets it via newUpdate; tests stub it directly.
	Update actionprofile.SetNameFunc
}

// UpdateArgs is kept as the CLI test-facing name for the action payload.
type UpdateArgs = actionprofile.SetNameRequest

// New builds the cobra command for "tg profile set-name".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "set-name <first>",
		Short: "Set profile first name (and optional last name)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.First = args[0]
			opts.LastSet = cmd.Flags().Changed("last")
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Update = newUpdate(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Last, "last", "", "Last name (pass \"\" to clear)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "first_name", "last_name"})
	return cmd
}

// Run validates options and dispatches the Update call.
func Run(ctx context.Context, opts *Options) error {
	if opts.Update == nil {
		return fmt.Errorf("%w: internal error: profile name update function is not configured", command.ErrPrecondition)
	}
	row, err := actionprofile.SetName(ctx, actionprofile.SetNameRequest{First: opts.First, Last: opts.Last, LastSet: opts.LastSet}, opts.Update)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.WriteProfileJSON(opts.IOStreams.Out, row)
}

// newUpdate returns the production Update closure that calls the Telegram API.
func newUpdate(f *runtime.Invocation) actionprofile.SetNameFunc {
	return func(ctx context.Context, a actionprofile.SetNameRequest) (output.ProfileRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ProfileRow{}, err
		}
		var row output.ProfileRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				var err error
				row, err = telegram.UpdateProfileName(ctx, api, a)
				return err
			})
		return row, err
	}
}
