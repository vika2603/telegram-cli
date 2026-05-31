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

// MuteTopicOptions holds the resolved flags and injected dependencies for the
// mute/unmute run.
type MuteTopicOptions struct {
	RawRef    string
	TopicID   int
	Unmute    bool
	Duration  string
	Until     string
	Forever   bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Mute      actionchat.MuteTopicFunc
}

// newMuteTopic builds "tg chat topics mute"; newUnmuteTopic builds the
// "unmute" counterpart. They share a factory (topicMuteCmd) to mirror the
// pin/unpin convention.
func newMuteTopic(f *runtime.Invocation, runF func(*MuteTopicOptions) error) *cobra.Command {
	return topicMuteCmd(f, runF, false, "mute", "Mute notifications for a forum topic")
}

func newUnmuteTopic(f *runtime.Invocation, runF func(*MuteTopicOptions) error) *cobra.Command {
	return topicMuteCmd(f, runF, true, "unmute", "Unmute notifications for a forum topic")
}

func topicMuteCmd(f *runtime.Invocation, runF func(*MuteTopicOptions) error, unmute bool, use, short string) *cobra.Command {
	opts := &MuteTopicOptions{Unmute: unmute}
	cmd := &cobra.Command{
		Use:               use + " <ref> <topic-id>",
		Short:             short,
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
			opts.Mute = newMuteTopicFn(f)
			return runMuteTopic(cmd.Context(), opts)
		},
	}
	if !unmute {
		cmd.Flags().StringVar(&opts.Duration, "duration", "", "Mute for a duration (e.g. 1h, 3d). Mutually exclusive with --until / --forever.")
		cmd.Flags().StringVar(&opts.Until, "until", "", "Mute until RFC3339 timestamp. Mutually exclusive with --duration / --forever.")
		cmd.Flags().BoolVar(&opts.Forever, "forever", false, "Mute indefinitely (default when no flag given).")
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"id"})
	return cmd
}

func runMuteTopic(ctx context.Context, opts *MuteTopicOptions) error {
	row, err := actionchat.MuteTopic(ctx, actionchat.MuteTopicRequest{
		RawRef:   opts.RawRef,
		TopicID:  opts.TopicID,
		Unmute:   opts.Unmute,
		Duration: opts.Duration,
		Until:    opts.Until,
		Forever:  opts.Forever,
	}, opts.Mute)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	verb := "muted"
	if opts.Unmute {
		verb = "unmuted"
	}
	_, err = fmt.Fprintf(opts.IOStreams.Out, "%s topic %d\n", verb, row.ID)
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
