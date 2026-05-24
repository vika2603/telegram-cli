// Package edit implements "tg msg edit <msg-ref>".
package edit

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
	RawMessageRef string
	Text          string
	Parse         string // "", "html", "markdown"

	IOStreams *ui.IOStreams
	Exporter  output.Exporter

	// Edit is the closure that performs the actual Telegram call. Production
	// code sets it via newEdit; tests stub it directly.
	Edit actionmessage.EditFunc
}

// New builds the cobra command for "tg msg edit".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "edit <msg-ref>",
		Short:             "Edit a message",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.MessageRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawMessageRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Edit = newEdit(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Text, "text", "", "New message body (required)")
	cmd.Flags().StringVar(&opts.Parse, "parse", "", "Parse mode: html | markdown")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "message_id", "chat_id", "date"})
	return cmd
}

// Run dispatches the normalized request and renders the result.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionmessage.Edit(ctx, actionmessage.EditRequest{
		RawMessageRef: opts.RawMessageRef,
		Text:          opts.Text,
		Parse:         opts.Parse,
	}, opts.Edit)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderSendResults(opts.IOStreams, []output.SendResultRow{row})
}

// newEdit returns the production Edit closure that calls the Telegram API.
func newEdit(f *runtime.Invocation) actionmessage.EditFunc {
	return func(ctx context.Context, q actionmessage.EditQuery) (output.SendResultRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.SendResultRow{}, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "msg.edit", q)
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
				row, err = telegram.EditMessage(ctx, api, res, q, f.IOStreams.ErrOut)
				return err
			})
		return row, err
	}
}
