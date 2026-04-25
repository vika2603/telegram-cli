// Package list implements "tg chat list".
package list

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	Limit        int
	ArchivedOnly bool
	PinnedOnly   bool
	Exporter     output.Exporter
	IOStreams    *ui.IOStreams
	Fetch        actionchat.ListFunc
}

// New builds the cobra command for "tg chat list".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List my dialogs",
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
	cmd.Flags().BoolVar(&opts.ArchivedOnly, "archived", false, "Only archived dialogs")
	cmd.Flags().BoolVar(&opts.PinnedOnly, "pinned", false, "Only pinned dialogs")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"peer", "title", "type", "unread", "pinned", "archived", "muted", "top_message", "last"})
	return cmd
}

// Run executes the list logic using opts.Fetch for data retrieval.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionchat.List(ctx, actionchat.ListRequest{
		Limit:        opts.Limit,
		ArchivedOnly: opts.ArchivedOnly,
		PinnedOnly:   opts.PinnedOnly,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderChatList(opts.IOStreams, rows)
}

func newFetch(f *runtime.Invocation) actionchat.ListFunc {
	return func(ctx context.Context, req actionchat.ListRequest) ([]output.ChatRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var out []output.ChatRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				out, err = telegram.ListChats(ctx, api, req)
				return err
			})
		return out, err
	}
}
