// Package block implements "tg contact block <ref>".
package block

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

	// Block is the closure that performs the actual Telegram call. Production
	// code sets it via newBlock; tests stub it directly.
	Block actioncontact.PeerFunc
}

// New builds the cobra command for "tg contact block".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "block <ref>",
		Short: "Block a user, bot, or channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			opts.Prompter = f.Prompter
			if runF != nil {
				return runF(opts)
			}
			opts.Block = newBlock(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run collects the raw request, delegates validation/mutation, and renders status.
func Run(ctx context.Context, opts *Options) error {
	if err := actioncontact.Block(ctx, actioncontact.PeerRequest{
		RawRef:   opts.RawRef,
		Yes:      opts.Yes,
		Prompter: opts.Prompter,
	}, opts.Block); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.IOStreams.Out, "blocked\t%s\n", opts.RawRef)
	return nil
}

// newBlock returns the production Block closure that calls the Telegram API.
func newBlock(f *runtime.Invocation) actioncontact.PeerFunc {
	return func(ctx context.Context, q actioncontact.PeerQuery) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		return f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				return telegram.BlockContact(ctx, api, res, q)
			})
	}
}
