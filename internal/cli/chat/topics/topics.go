// Package topics implements "tg chat topics" (list) and its "create" child.
package topics

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// ListOptions holds the resolved flags and injected dependencies for the
// list run.
type ListOptions struct {
	RawRef    string
	Q         string
	Limit     int
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Fetch     actionchat.TopicsFunc
}

// New builds the "tg chat topics" command: listing by default, with a
// "create" subcommand.
func New(f *runtime.Invocation, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{}
	cmd := &cobra.Command{
		Use:               "topics <ref>",
		Short:             "List forum topics of a supergroup",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch(f)
			return runList(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Q, "q", "", "Substring filter on topic titles")
	cmd.Flags().IntVar(&opts.Limit, "limit", 100, "Max topics to list")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"id", "title", "icon_color", "icon_emoji_id", "top_message", "unread_count", "closed", "hidden", "pinned"})
	cmd.AddCommand(newCreate(f, nil))
	return cmd
}

func runList(ctx context.Context, opts *ListOptions) error {
	rows, err := actionchat.Topics(ctx, actionchat.TopicsRequest{
		RawRef: opts.RawRef,
		Q:      opts.Q,
		Limit:  opts.Limit,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderTopicList(opts.IOStreams, rows)
}

func newFetch(f *runtime.Invocation) actionchat.TopicsFunc {
	return func(ctx context.Context, q actionchat.TopicsQuery) ([]output.TopicRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.TopicRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.ListForumTopics(ctx, api, res, q)
				return err
			})
		return rows, err
	}
}
