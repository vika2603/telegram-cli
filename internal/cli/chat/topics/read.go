package topics

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

// ReadTopicOptions holds the resolved flags and injected dependencies for the
// topic read run.
type ReadTopicOptions struct {
	RawRef    string
	TopicID   int
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Read      actionchat.ReadTopicFunc
}

func newReadTopic(f *runtime.Invocation, runF func(*ReadTopicOptions) error) *cobra.Command {
	opts := &ReadTopicOptions{}
	cmd := &cobra.Command{
		Use:               "read <ref> <topic-id>",
		Short:             "Mark a forum topic as read",
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
			opts.Read = newReadTopicFn(f)
			return runReadTopic(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"id"})
	return cmd
}

func runReadTopic(ctx context.Context, opts *ReadTopicOptions) error {
	row, err := actionchat.ReadTopic(ctx, actionchat.ReadTopicRequest{
		RawRef:  opts.RawRef,
		TopicID: opts.TopicID,
	}, opts.Read)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	_, err = fmt.Fprintf(opts.IOStreams.Out, "read topic %d\n", row.ID)
	return err
}

func newReadTopicFn(f *runtime.Invocation) actionchat.ReadTopicFunc {
	return func(ctx context.Context, q actionchat.ReadTopicQuery) (output.TopicRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.TopicRow{}, err
		}
		var row output.TopicRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.ReadForumTopic(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
