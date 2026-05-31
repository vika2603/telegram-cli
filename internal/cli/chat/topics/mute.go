package topics

import (
	"context"
	"fmt"
	"math"
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

// MuteTopicOptions holds the resolved flags and injected dependencies for the
// mute/unmute run.
type MuteTopicOptions struct {
	RawRef    string
	TopicID   int
	MuteUntil int
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Mute      actionchat.MuteTopicFunc
}

// newMuteTopic builds "tg chat topics mute"; newUnmuteTopic builds the
// "unmute" counterpart. They share a factory (topicMuteCmd) to mirror the
// pin/unpin convention.
func newMuteTopic(f *runtime.Invocation, runF func(*MuteTopicOptions) error) *cobra.Command {
	return topicMuteCmd(f, runF, math.MaxInt32, "mute", "Mute notifications for a forum topic")
}

func newUnmuteTopic(f *runtime.Invocation, runF func(*MuteTopicOptions) error) *cobra.Command {
	return topicMuteCmd(f, runF, 0, "unmute", "Unmute notifications for a forum topic")
}

func topicMuteCmd(f *runtime.Invocation, runF func(*MuteTopicOptions) error, muteUntil int, use, short string) *cobra.Command {
	opts := &MuteTopicOptions{MuteUntil: muteUntil}
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
			opts.Mute = newMuteTopicFn(f)
			return runMuteTopic(cmd.Context(), opts, use)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"id"})
	return cmd
}

func runMuteTopic(ctx context.Context, opts *MuteTopicOptions, verb string) error {
	row, err := actionchat.MuteTopic(ctx, actionchat.MuteTopicRequest{
		RawRef:    opts.RawRef,
		TopicID:   opts.TopicID,
		MuteUntil: opts.MuteUntil,
	}, opts.Mute)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	_, err = fmt.Fprintf(opts.IOStreams.Out, "%sd topic %d\n", verb, row.ID)
	return err
}

func newMuteTopicFn(f *runtime.Invocation) actionchat.MuteTopicFunc {
	return func(ctx context.Context, q actionchat.MuteTopicQuery) (output.TopicRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.TopicRow{}, err
		}
		var row output.TopicRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.MuteForumTopic(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
