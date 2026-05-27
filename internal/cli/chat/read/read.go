// Package read implements "tg chat mark-read <ref>".
package read

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawRef string
	MaxID  int

	IOStreams *ui.IOStreams

	Read actionchat.ReadFunc
}

// New builds the cobra command for "tg chat mark-read".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "mark-read <ref>",
		Short:             "Mark chat as read up to a message id (default: all)",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Read = newRead(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().IntVar(&opts.MaxID, "max-id", 0, "Mark as read up to this message id (0 = all)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run validates options and dispatches to opts.Read.
func Run(ctx context.Context, opts *Options) error {
	if err := actionchat.Read(ctx, actionchat.ReadRequest{
		RawRef: opts.RawRef,
		MaxID:  opts.MaxID,
	}, opts.Read); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.IOStreams.Out, "read\t%s\n", opts.RawRef)
	return nil
}

func newRead(f *runtime.Invocation) actionchat.ReadFunc {
	return func(ctx context.Context, q actionchat.ReadQuery) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			_, err := cl.Call(ctx, "chat.mark_read", q)
			return err
		}
		return f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				return telegram.ReadChat(ctx, api, res, q)
			})
	}
}
