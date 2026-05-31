package topic

import (
	"context"
	"fmt"
	"strconv"

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

// InfoTopicOptions holds the resolved flags and injected dependencies for the
// topic info run.
type InfoTopicOptions struct {
	RawRef    string
	TopicID   int
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Info      actionchat.TopicInfoFunc
}

func newTopicInfo(f *runtime.Invocation, runF func(*InfoTopicOptions) error) *cobra.Command {
	opts := &InfoTopicOptions{}
	cmd := &cobra.Command{
		Use:               "info <ref> <topic-id>",
		Short:             "Show details for a single forum topic",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			id, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("%w: topic id must be a positive integer", command.ErrUsage)
			}
			opts.TopicID = id
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Info = newTopicInfoFn(f)
			return runTopicInfo(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"id", "title", "icon_color", "icon_emoji_id", "top_message", "unread_count", "closed", "hidden", "pinned"})
	return cmd
}

func runTopicInfo(ctx context.Context, opts *InfoTopicOptions) error {
	row, err := actionchat.InfoTopic(ctx, actionchat.TopicInfoRequest{
		RawRef:  opts.RawRef,
		TopicID: opts.TopicID,
	}, opts.Info)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderTopic(opts.IOStreams, row)
}

func newTopicInfoFn(f *runtime.Invocation) actionchat.TopicInfoFunc {
	return func(ctx context.Context, q actionchat.TopicInfoQuery) (output.TopicRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.TopicRow{}, err
		}
		var row output.TopicRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.GetForumTopicByID(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
