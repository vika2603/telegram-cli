// Package schedulecancel implements "tg msg schedule-cancel <ref> <id> [<id>...]".
package schedulecancel

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawRef string
	IDs    []int
	Yes    bool

	Prompter  ui.Prompter
	IOStreams *ui.IOStreams

	// Cancel is the closure that performs the actual Telegram call. Production
	// code sets it via newCancel; tests stub it directly.
	Cancel actionmessage.CancelScheduledFunc
}

// New builds the cobra command for "tg msg schedule-cancel".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "schedule-cancel <ref> <id> [<id>...]",
		Short: "Cancel scheduled messages",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			ids := make([]int, 0, len(args)-1)
			for _, s := range args[1:] {
				n, err := strconv.Atoi(s)
				if err != nil {
					return fmt.Errorf("%w: message id must be an integer: %q", command.ErrUsage, s)
				}
				ids = append(ids, n)
			}
			opts.IDs = ids
			opts.IOStreams = f.IOStreams
			opts.Prompter = f.Prompter
			if runF != nil {
				return runF(opts)
			}
			opts.Cancel = newCancel(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run dispatches the normalized request and renders the result.
func Run(ctx context.Context, opts *Options) error {
	result, err := actionmessage.CancelScheduled(ctx, actionmessage.CancelScheduledRequest{
		RawRef:   opts.RawRef,
		IDs:      opts.IDs,
		Yes:      opts.Yes,
		Prompter: opts.Prompter,
	}, opts.Cancel)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.IOStreams.Out, "cancelled\t%d\n", result.Count)
	return nil
}

// newCancel returns the production Cancel closure that calls the Telegram API.
func newCancel(f *runtime.Invocation) actionmessage.CancelScheduledFunc {
	return func(ctx context.Context, q actionmessage.CancelScheduledQuery) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		return f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				return telegram.CancelScheduledMessages(ctx, api, res, q)
			})
	}
}
