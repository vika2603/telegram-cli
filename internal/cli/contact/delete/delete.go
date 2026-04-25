// Package deletecmd implements "tg contact delete <ref>".
package deletecmd

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawRef string
	Yes    bool

	IOStreams *ui.IOStreams
	Prompter  ui.Prompter

	// Delete is the closure that performs the actual Telegram call. Production
	// code sets it via newDelete; tests stub it directly.
	Delete actioncontact.PeerFunc
}

// New builds the cobra command for "tg contact delete".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "delete <ref>",
		Short: "Delete a contact",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			opts.Prompter = f.Prompter
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

// Run collects the raw request, delegates validation/mutation, and renders status.
func Run(ctx context.Context, opts *Options) error {
	if err := actioncontact.Delete(ctx, actioncontact.PeerRequest{
		RawRef:   opts.RawRef,
		Yes:      opts.Yes,
		Prompter: opts.Prompter,
	}, opts.Delete); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.IOStreams.Out, "deleted\t%s\n", opts.RawRef)
	return nil
}

// newDelete returns the production Delete closure that calls the Telegram API.
func newDelete(f *runtime.Invocation) actioncontact.PeerFunc {
	return func(ctx context.Context, q actioncontact.PeerQuery) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		return f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				return telegram.DeleteContact(ctx, api, res, q)
			})
	}
}
