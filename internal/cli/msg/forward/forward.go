// Package forward implements "tg msg forward <msg-ref> [<msg-ref>...] --to <dst>".
package forward

import (
	"context"
	"encoding/json"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawMessageRefs []string
	RawTo          string
	Silent         bool
	RandomID       int64

	Exporter  output.Exporter
	IOStreams *ui.IOStreams

	// Forward is the closure that performs the actual Telegram call. Production
	// code sets it via newForward; tests stub it directly.
	Forward actionmessage.ForwardFunc
}

// New builds the cobra command for "tg msg forward".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "forward <msg-ref> [<msg-ref>...]",
		Short:             "Forward messages to another chat",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: complete.MessageRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawMessageRefs = args
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Forward = newForward(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.RawTo, "to", "", "Destination chat <ref> (required)")
	_ = cmd.RegisterFlagCompletionFunc("to", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return complete.PeerRefs(f)(cmd, args, toComplete)
	})
	cmd.Flags().BoolVar(&opts.Silent, "silent", false, "Do not notify recipients")
	cmd.Flags().Int64Var(&opts.RandomID, "random-id", 0, "Idempotency key (int64): reusing it on retry dedupes the forward server-side")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "message_id", "chat_id", "date"})
	return cmd
}

// Run dispatches the normalized request and renders the result.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionmessage.Forward(ctx, actionmessage.ForwardRequest{
		RawMessageRefs: opts.RawMessageRefs,
		RawTo:          opts.RawTo,
		Silent:         opts.Silent,
		RandomID:       opts.RandomID,
	}, opts.Forward)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderSendResults(opts.IOStreams, []output.SendResultRow{row})
}

// newForward returns the production Forward closure that calls the Telegram API.
func newForward(f *runtime.Invocation) actionmessage.ForwardFunc {
	return func(ctx context.Context, q actionmessage.ForwardQuery) (output.SendResultRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.SendResultRow{}, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "msg.forward", q)
			if err != nil {
				return output.SendResultRow{}, err
			}
			var row output.SendResultRow
			if err := json.Unmarshal(raw, &row); err != nil {
				return output.SendResultRow{}, err
			}
			return row, nil
		}

		var row output.SendResultRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.ForwardMessages(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
