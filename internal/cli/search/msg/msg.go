// Package msg implements "tg search msg <query>".
package msg

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionsearch "github.com/vika2603/telegram-cli/internal/action/search"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Query          string
	In             string
	Filter         string
	From           string
	MinDate        string
	MaxDate        string
	BroadcastsOnly bool
	GroupsOnly     bool
	UsersOnly      bool
	Missed         bool
	Limit          int
	Order          string
	Exporter       output.Exporter
	IOStreams      *ui.IOStreams
	Fetch          actionsearch.MessageFunc
}

// New builds the cobra command for "tg search msg".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "msg <query>",
		Short: "Search messages (global by default, --in narrows to one chat)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Query = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.In, "in", "", "Narrow to one chat (ref)")
	cmd.Flags().StringVar(&opts.Filter, "filter", "",
		"Content filter: photos|video|document|voice|music|gif|url|pinned|geo|my-mentions|round-video|round-voice|phone-calls|chat-photos|contacts|poll|photo-video")
	cmd.Flags().StringVar(&opts.From, "from", "", "Filter by sender (in-chat only)")
	cmd.Flags().StringVar(&opts.MinDate, "min-date", "", "RFC3339 lower bound")
	cmd.Flags().StringVar(&opts.MaxDate, "max-date", "", "RFC3339 upper bound")
	cmd.Flags().BoolVar(&opts.BroadcastsOnly, "broadcasts-only", false, "Global-search only: channels")
	cmd.Flags().BoolVar(&opts.GroupsOnly, "groups-only", false, "Global-search only: groups")
	cmd.Flags().BoolVar(&opts.UsersOnly, "users-only", false, "Global-search only: 1:1 chats")
	cmd.Flags().BoolVar(&opts.Missed, "missed", false, "With --filter phone-calls: missed only")
	cmd.Flags().IntVar(&opts.Limit, "limit", 20, "Max results (cap 1000)")
	cmd.Flags().StringVar(&opts.Order, "order", "desc", "asc|desc")

	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"message_id", "chat_id", "chat_title", "chat_kind", "date", "from_id", "text", "media_kind"})
	return cmd
}

// Run executes the search msg logic using opts.Fetch for data retrieval.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionsearch.Message(ctx, actionsearch.MessageRequest{
		Query:          opts.Query,
		In:             opts.In,
		Filter:         opts.Filter,
		From:           opts.From,
		MinDate:        opts.MinDate,
		MaxDate:        opts.MaxDate,
		BroadcastsOnly: opts.BroadcastsOnly,
		GroupsOnly:     opts.GroupsOnly,
		UsersOnly:      opts.UsersOnly,
		Missed:         opts.Missed,
		Limit:          opts.Limit,
		Order:          opts.Order,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderSearchMsg(opts.IOStreams, rows)
}

func newFetch(f *runtime.Invocation) actionsearch.MessageFunc {
	return func(ctx context.Context, q actionsearch.MessageQuery) ([]output.SearchMsgRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.SearchMsgRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.SearchMessages(ctx, api, res, q)
				return err
			})
		return rows, err
	}
}
