// Package pin implements "tg msg pin" and the shared core for "tg msg unpin".
package pin

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawMessageRef string
	Silent        bool
	Unpin         bool

	IOStreams *ui.IOStreams

	// Do is the closure that performs the actual Telegram call. Production
	// code sets it via newDo; tests stub it directly.
	Do actionmessage.PinFunc
}

// New builds the cobra command for "tg msg pin".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return newCmd(f, runF, "pin", "Pin a message", false)
}

// NewUnpin builds the cobra command for "tg msg unpin".
func NewUnpin(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return newCmd(f, runF, "unpin", "Unpin a message", true)
}

// newCmd is the shared cobra wiring.
func newCmd(f *runtime.Invocation, runF func(*Options) error, use, short string, unpin bool) *cobra.Command {
	opts := &Options{Unpin: unpin}
	cmd := &cobra.Command{
		Use:               use + " <msg-ref>",
		Short:             short,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.MessageRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawMessageRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Silent, "silent", false, "Do not notify recipients")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run dispatches the normalized request and renders the result.
func Run(ctx context.Context, opts *Options) error {
	result, err := actionmessage.Pin(ctx, actionmessage.PinRequest{
		RawMessageRef: opts.RawMessageRef,
		Silent:        opts.Silent,
		Unpin:         opts.Unpin,
	}, opts.Do)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(opts.IOStreams.Out, "%s\t%d\n", result.Verb, result.MessageID)
	return nil
}

// newDo returns the production Do closure that calls the Telegram API.
// Used by both "tg msg pin" and "tg msg unpin" (q.Unpin discriminates).
func newDo(f *runtime.Invocation) actionmessage.PinFunc {
	return func(ctx context.Context, q actionmessage.PinQuery) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			_, err := cl.Call(ctx, "msg.pin", q)
			return err
		}
		return f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				return telegram.PinMessage(ctx, api, res, q)
			})
	}
}
