// Package info implements "tg msg info <msg-ref>".
package info

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds flags/deps for `msg info`.
type Options struct {
	RawMessageRef string
	Exporter      output.Exporter
	IOStreams     *ui.IOStreams
	Do            actionmessage.InfoFunc
}

// New builds the cobra command for "tg msg info".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "info <msg-ref>",
		Short: "Show one message's details (expands poll content)",
		Args:  cobra.ExactArgs(1),
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
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"id", "ref", "date", "edit_date", "from", "text", "reply_to", "forward", "reactions", "media", "album", "views", "is_pinned"})
	return cmd
}

// Run dispatches and renders.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionmessage.MessageInfo(ctx, actionmessage.InfoRequest{RawMessageRef: opts.RawMessageRef}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderMessageDetail(opts.IOStreams, row)
}

func newDo(f *runtime.Invocation) actionmessage.InfoFunc {
	return func(ctx context.Context, q actionmessage.InfoQuery) (output.MessageRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.MessageRow{}, err
		}
		var row output.MessageRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, resolver *peer.Resolver) error {
				row, err = telegram.MessageDetail(ctx, api, resolver, q)
				return err
			})
		return row, err
	}
}
