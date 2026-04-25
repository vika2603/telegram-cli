// Package chat implements "tg search chat <query>".
package chat

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
	Query     string
	Kind      string
	MyOnly    bool
	Limit     int
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Fetch     actionsearch.ChatFunc
}

// New builds the cobra command for "tg search chat".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "chat <query>",
		Short: "Search chats, users, channels, and bots",
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
	cmd.Flags().StringVar(&opts.Kind, "kind", "", "Narrow by kind: user|group|channel|bot")
	cmd.Flags().BoolVar(&opts.MyOnly, "my-only", false, "Only results from my contacts/dialogs")
	cmd.Flags().IntVar(&opts.Limit, "limit", 20, "Max results (server-capped)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"peer", "title", "type", "source"})
	return cmd
}

// Run executes the search chat logic using opts.Fetch for data retrieval.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionsearch.Chat(ctx, actionsearch.ChatRequest{
		Query:  opts.Query,
		Kind:   opts.Kind,
		MyOnly: opts.MyOnly,
		Limit:  opts.Limit,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderSearchChat(opts.IOStreams, rows)
}

func newFetch(f *runtime.Invocation) actionsearch.ChatFunc {
	return func(ctx context.Context, q actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.SearchChatRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				rows, err = telegram.SearchChats(ctx, api, q)
				return err
			})
		return rows, err
	}
}
