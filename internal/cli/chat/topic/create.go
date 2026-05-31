package topic

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

// CreateOptions holds the resolved flags and injected dependencies for the
// create run.
type CreateOptions struct {
	RawRef      string
	Title       string
	IconColor   int
	IconEmojiID int64
	RandomID    int64
	Exporter    output.Exporter
	IOStreams   *ui.IOStreams
	Create      actionchat.CreateTopicFunc
}

func newCreate(f *runtime.Invocation, runF func(*CreateOptions) error) *cobra.Command {
	opts := &CreateOptions{}
	cmd := &cobra.Command{
		Use:               "create <ref> <title>",
		Short:             "Create a forum topic in a supergroup",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.Title = args[1]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Create = newCreateFn(f)
			return runCreate(cmd.Context(), opts)
		},
	}
	cmd.Flags().IntVar(&opts.IconColor, "icon-color", 0, "Topic icon color (RGB int, e.g. 0x6FB9F0)")
	cmd.Flags().Int64Var(&opts.IconEmojiID, "icon-emoji", 0, "Custom emoji id for the topic icon")
	cmd.Flags().Int64Var(&opts.RandomID, "random-id", 0, "Idempotency key (int64): reusing it on retry avoids creating a duplicate topic")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"id", "title", "icon_color", "icon_emoji_id"})
	return cmd
}

func runCreate(ctx context.Context, opts *CreateOptions) error {
	row, err := actionchat.CreateTopic(ctx, actionchat.CreateTopicRequest{
		RawRef:      opts.RawRef,
		Title:       opts.Title,
		IconColor:   opts.IconColor,
		IconEmojiID: opts.IconEmojiID,
		RandomID:    opts.RandomID,
	}, opts.Create)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderTopic(opts.IOStreams, row)
}

func newCreateFn(f *runtime.Invocation) actionchat.CreateTopicFunc {
	return func(ctx context.Context, q actionchat.CreateTopicQuery) (output.TopicRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.TopicRow{}, err
		}
		var row output.TopicRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.CreateForumTopic(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
