// Package link implements "tg msg link <msg-ref>".
package link

import (
	"context"
	"encoding/json"
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
	IOStreams     *ui.IOStreams
	Resolve       actionmessage.LinkResolveFunc
}

// New builds the cobra command for "tg msg link".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "link <msg-ref>",
		Short:             "Emit the t.me link for a message",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.MessageRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawMessageRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Resolve = newResolve(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	return cmd
}

// Run emits a t.me link for the given message.
func Run(ctx context.Context, opts *Options) error {
	url, err := actionmessage.Link(ctx, actionmessage.LinkRequest{
		RawMessageRef: opts.RawMessageRef,
	}, opts.Resolve)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(opts.IOStreams.Out, url)
	return err
}

func newResolve(f *runtime.Invocation) actionmessage.LinkResolveFunc {
	return func(ctx context.Context, q actionmessage.LinkQuery) (actionmessage.LinkPeer, error) {
		acct, err := f.Account("")
		if err != nil {
			return actionmessage.LinkPeer{}, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "msg.link", q)
			if err != nil {
				return actionmessage.LinkPeer{}, err
			}
			var out actionmessage.LinkPeer
			if err := json.Unmarshal(raw, &out); err != nil { //nolint:musttag
				return actionmessage.LinkPeer{}, err
			}
			return out, nil
		}

		var out actionmessage.LinkPeer
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, _ *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				out, err = telegram.ResolveMessageLinkPeer(ctx, res, q)
				return err
			})
		return out, err
	}
}
