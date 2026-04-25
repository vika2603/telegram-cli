// Package digest implements the top-level "tg digest" command.
package digest

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds resolved flags and injectable dependencies for Run.
type Options struct {
	RawRef    string
	Limit     int
	MinDate   string
	MaxDate   string
	Order     string
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Fetch     actionmessage.ListFunc
}

// New builds the cobra command for "tg digest".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "digest <ref>",
		Short:             "Show a compact message digest for a chat",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
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
	cmd.Flags().IntVar(&opts.Limit, "limit", 50, "Max messages (cap 1000)")
	cmd.Flags().StringVar(&opts.MinDate, "min-date", "", "RFC3339 lower bound")
	cmd.Flags().StringVar(&opts.MaxDate, "max-date", "", "RFC3339 upper bound")
	cmd.Flags().StringVar(&opts.Order, "order", "desc", "asc|desc")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"ref", "id", "date", "from", "text", "media", "reply_to", "views", "is_pinned"})
	return cmd
}

// Run executes the digest query and renders rows.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionmessage.List(ctx, actionmessage.ListRequest{
		RawRef:  opts.RawRef,
		Limit:   opts.Limit,
		MinDate: opts.MinDate,
		MaxDate: opts.MaxDate,
		Order:   opts.Order,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderDigest(opts.IOStreams, rows)
}

func newFetch(f *runtime.Invocation) actionmessage.ListFunc {
	return func(ctx context.Context, q actionmessage.ListQuery) ([]output.MessageRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.MessageRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.ListMessages(ctx, api, res, q)
				if err == nil {
					recordRecentMessages(res.Store(), q.Ref.String(), rows)
				}
				return err
			})
		return rows, err
	}
}

func recordRecentMessages(store *account.PeerStore, peerRef string, rows []output.MessageRow) {
	if store == nil {
		return
	}
	for _, row := range rows {
		if row.Ref == "" {
			continue
		}
		_ = store.RecordRecentMessage(account.RecentMessage{
			Ref:       row.Ref,
			PeerRef:   peerRef,
			MessageID: row.ID,
			Date:      row.Date,
			Text:      row.Text,
		})
	}
}
