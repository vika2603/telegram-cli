// Package poll implements "tg msg poll <ref> <question> <option>...".
package poll

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds flags/deps for `msg poll`.
type Options struct {
	RawRef      string
	Question    string
	PollOptions []string
	Multiple    bool
	Public      bool
	Correct     int
	Explanation string
	Exporter    output.Exporter
	IOStreams   *ui.IOStreams
	Do          actionmessage.PollFunc
}

// New builds the cobra command for "tg msg poll".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "poll <ref> <question> <option> <option> [option...]",
		Short:             "Send a poll (or a quiz with --correct)",
		Args:              cobra.MinimumNArgs(4),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.Question = args[1]
			opts.PollOptions = args[2:]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Multiple, "multiple", false, "Allow selecting multiple answers")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Public voters (default anonymous)")
	cmd.Flags().IntVar(&opts.Correct, "correct", 0, "Quiz mode: 1-based index of the correct option")
	cmd.Flags().StringVar(&opts.Explanation, "explanation", "", "Quiz solution shown after answering (requires --correct)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "message_id", "chat_id", "date"})
	return cmd
}

// Run validates and dispatches.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionmessage.Poll(ctx, actionmessage.PollRequest{
		RawRef:      opts.RawRef,
		Question:    opts.Question,
		Options:     opts.PollOptions,
		Multiple:    opts.Multiple,
		Public:      opts.Public,
		Correct:     opts.Correct,
		Explanation: opts.Explanation,
	}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderSendResults(opts.IOStreams, rows)
}

func newDo(f *runtime.Invocation) actionmessage.PollFunc {
	return func(ctx context.Context, q actionmessage.PollQuery) ([]output.SendResultRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.SendResultRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.SendPoll(ctx, api, res, q)
				return err
			})
		return rows, err
	}
}
