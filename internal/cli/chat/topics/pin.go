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

// PinTopicOptions holds the resolved flags and injected dependencies for the
// pin run.
type PinTopicOptions struct {
	RawRef    string
	TopicID   int
	Unpin     bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Pin       actionchat.PinTopicFunc
}

// newPinTopic builds "tg chat topics pin"; newUnpinTopic builds the "unpin"
// counterpart. They are separate commands (not one with a --unpin flag) to
// match the repo's toggle convention (msg pin/unpin, chat mute/unmute, …).
func newPinTopic(f *runtime.Invocation, runF func(*PinTopicOptions) error) *cobra.Command {
	return topicPinCmd(f, runF, false, "pin", "Pin a forum topic")
}

func newUnpinTopic(f *runtime.Invocation, runF func(*PinTopicOptions) error) *cobra.Command {
	return topicPinCmd(f, runF, true, "unpin", "Unpin a forum topic")
}

func topicPinCmd(f *runtime.Invocation, runF func(*PinTopicOptions) error, unpin bool, use, short string) *cobra.Command {
	opts := &PinTopicOptions{Unpin: unpin}
	cmd := &cobra.Command{
		Use:   use + " <ref> <topic-id>",
		Short: short,
		Args:  cobra.ExactArgs(2),
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
			opts.Pin = newPinTopicFn(f)
			return runPinTopic(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"id", "title", "closed", "hidden", "pinned"})
	return cmd
}

func runPinTopic(ctx context.Context, opts *PinTopicOptions) error {
	row, err := actionchat.PinTopic(ctx, actionchat.PinTopicRequest{
		RawRef:  opts.RawRef,
		TopicID: opts.TopicID,
		Unpin:   opts.Unpin,
	}, opts.Pin)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	verb := "pinned"
	if opts.Unpin {
		verb = "unpinned"
	}
	_, err = fmt.Fprintf(opts.IOStreams.Out, "%s topic %d\n", verb, row.ID)
	return err
}

func newPinTopicFn(f *runtime.Invocation) actionchat.PinTopicFunc {
	return func(ctx context.Context, q actionchat.PinTopicQuery) (output.TopicRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.TopicRow{}, err
		}
		var row output.TopicRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.PinForumTopic(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
