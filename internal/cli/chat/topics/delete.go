package topics

import (
	"context"
	"fmt"
	"strconv"

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

// DeleteTopicOptions holds the resolved flags and injected dependencies for
// the topic delete run.
type DeleteTopicOptions struct {
	RawRef    string
	TopicID   int
	Yes       bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Prompter  ui.Prompter
	Delete    actionchat.DeleteTopicFunc
}

func newDeleteTopic(f *runtime.Invocation, runF func(*DeleteTopicOptions) error) *cobra.Command {
	opts := &DeleteTopicOptions{}
	cmd := &cobra.Command{
		Use:   "delete <ref> <topic-id>",
		Short: "Delete a forum topic and its message history",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			id, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("%w: topic id must be a positive integer", command.ErrUsage)
			}
			opts.TopicID = id
			opts.IOStreams = f.IOStreams
			opts.Prompter = f.Prompter
			if runF != nil {
				return runF(opts)
			}
			opts.Delete = newDeleteTopicFn(f)
			return runDeleteTopic(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"id", "title", "closed", "hidden"})
	return cmd
}

func runDeleteTopic(ctx context.Context, opts *DeleteTopicOptions) error {
	row, err := actionchat.DeleteTopic(ctx, actionchat.DeleteTopicRequest{
		RawRef:   opts.RawRef,
		TopicID:  opts.TopicID,
		Yes:      opts.Yes,
		Prompter: opts.Prompter,
	}, opts.Delete)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	_, err = fmt.Fprintf(opts.IOStreams.Out, "deleted topic %d\n", row.ID)
	return err
}

func newDeleteTopicFn(f *runtime.Invocation) actionchat.DeleteTopicFunc {
	return func(ctx context.Context, q actionchat.DeleteTopicQuery) error {
		acct, err := f.Account("")
		if err != nil {
			return err
		}
		return f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				return telegram.DeleteForumTopic(ctx, api, res, q)
			})
	}
}
