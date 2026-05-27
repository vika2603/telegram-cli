// Package schedulelist implements "tg msg schedule-list <ref>".
package schedulelist

import (
	"context"
	"encoding/json"

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

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawRef    string
	IOStreams *ui.IOStreams
	Exporter  output.Exporter

	// Fetch is the closure that performs the actual Telegram call. Production
	// code sets it via newFetch; tests stub it directly.
	Fetch actionmessage.ScheduledListFunc
}

// New builds the cobra command for "tg msg schedule-list".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "schedule-list <ref>",
		Short: "List scheduled messages in a chat",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"id", "date", "scheduled_for", "text", "from_id"})
	return cmd
}

// Run dispatches the normalized request and renders the result.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionmessage.ScheduledList(ctx, actionmessage.ScheduledListRequest{
		RawRef: opts.RawRef,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderScheduled(opts.IOStreams, rows)
}

// newFetch returns the production Fetch closure that calls the Telegram API.
func newFetch(f *runtime.Invocation) actionmessage.ScheduledListFunc {
	return func(ctx context.Context, q actionmessage.ScheduledListQuery) ([]output.ScheduledMessageRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "msg.schedule_list", q)
			if err != nil {
				return nil, err
			}
			var rows []output.ScheduledMessageRow
			if err := json.Unmarshal(raw, &rows); err != nil {
				return nil, err
			}
			return rows, nil
		}

		var rows []output.ScheduledMessageRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.ListScheduledMessages(ctx, api, res, q)
				return err
			})
		return rows, err
	}
}
