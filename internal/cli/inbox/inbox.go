// Package inbox implements the top-level "tg inbox" command.
package inbox

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds resolved flags and injectable dependencies for Run.
type Options struct {
	Limit        int
	UnreadOnly   bool
	ArchivedOnly bool
	Exporter     output.Exporter
	IOStreams    *ui.IOStreams
	Fetch        actionchat.ListFunc
}

// New builds the cobra command for "tg inbox".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "Show recent dialogs with unread counts and last messages",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().IntVar(&opts.Limit, "limit", 30, "Maximum number of dialogs (cap 1000)")
	cmd.Flags().BoolVar(&opts.UnreadOnly, "unread", false, "Only dialogs with unread messages")
	cmd.Flags().BoolVar(&opts.ArchivedOnly, "archived", false, "Only archived dialogs")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"peer", "title", "type", "unread", "pinned", "archived", "muted", "top_message", "last"})
	return cmd
}

// Run executes the inbox query and renders rows.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionchat.List(ctx, actionchat.ListRequest{
		Limit:        opts.Limit,
		ArchivedOnly: opts.ArchivedOnly,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.UnreadOnly {
		rows = filterUnread(rows)
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderChatList(opts.IOStreams, rows)
}

func filterUnread(rows []output.ChatRow) []output.ChatRow {
	if len(rows) == 0 {
		return rows
	}
	out := rows[:0]
	for _, row := range rows {
		if row.UnreadCount > 0 {
			out = append(out, row)
		}
	}
	return out
}

func newFetch(f *runtime.Invocation) actionchat.ListFunc {
	return func(ctx context.Context, req actionchat.ListRequest) ([]output.ChatRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.ChatRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.ListChats(ctx, api, req)
				if err == nil {
					recordInboxPeers(res.Store(), rows)
				}
				return err
			})
		return rows, err
	}
}

func recordInboxPeers(store *account.PeerStore, rows []output.ChatRow) {
	if store == nil {
		return
	}
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		if row.Ref == "" {
			continue
		}
		_ = store.RecordRecentPeer(account.RecentPeer{
			Ref:      row.Ref,
			ID:       row.ID,
			Kind:     row.Kind,
			Title:    row.Title,
			Username: row.Username,
		})
	}
}
