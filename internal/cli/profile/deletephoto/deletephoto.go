// Package deletephoto implements "tg profile delete-photo".
package deletephoto

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionprofile "github.com/vika2603/telegram-cli/internal/action/profile"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Yes bool

	F         *runtime.Invocation
	IOStreams *ui.IOStreams

	// Delete is the closure that performs the actual Telegram call. Production
	// code sets it via newDelete; tests stub it directly.
	Delete actionprofile.DeletePhotoFunc
}

// New builds the cobra command for "tg profile delete-photo".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{F: f}
	cmd := &cobra.Command{
		Use:   "delete-photo",
		Short: "Remove my current profile photo",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Delete = newDelete(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run validates options, prompts for confirmation, and dispatches the Delete call.
func Run(ctx context.Context, opts *Options) error {
	if opts.Delete == nil {
		return fmt.Errorf("%w: internal error: profile photo delete function is not configured", command.ErrPrecondition)
	}
	if err := actionprofile.DeletePhoto(ctx, actionprofile.DeletePhotoRequest{Yes: opts.Yes, Prompter: opts.F.Prompter}, opts.Delete); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(opts.IOStreams.Out, "deleted")
	return nil
}

// newDelete returns the production Delete closure that calls the Telegram API.
func newDelete(f *runtime.Invocation) func(context.Context) error {
	return func(ctx context.Context) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		return f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				return telegram.DeleteProfilePhoto(ctx, api)
			})
	}
}
