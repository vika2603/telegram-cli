// Package vote implements "tg msg vote <msg-ref> [option...]".
package vote

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds flags/deps for `msg vote`.
type Options struct {
	RawMessageRef string
	Options       []int
	Retract       bool
	Exporter      output.Exporter
	IOStreams     *ui.IOStreams
	Do            actionmessage.VoteFunc
}

// New builds the cobra command for "tg msg vote".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "vote <msg-ref> [option-number...]",
		Short: "Vote on a poll, or show it when no option is given",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawMessageRef = args[0]
			nums, err := parseOptionNumbers(args[1:])
			if err != nil {
				return err
			}
			opts.Options = nums
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Retract, "retract", false, "Retract your vote")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"question", "options", "total_voters", "multiple", "quiz", "public", "closed"})
	return cmd
}

func parseOptionNumbers(args []string) ([]int, error) {
	nums := make([]int, 0, len(args))
	for _, a := range args {
		n, err := strconv.Atoi(a)
		if err != nil {
			return nil, fmt.Errorf("%w: option must be a number, got %q", command.ErrUsage, a)
		}
		nums = append(nums, n)
	}
	return nums, nil
}

// Run validates and dispatches.
func Run(ctx context.Context, opts *Options) error {
	info, err := actionmessage.Vote(ctx, actionmessage.VoteRequest{
		RawMessageRef: opts.RawMessageRef,
		Options:       opts.Options,
		Retract:       opts.Retract,
	}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, info)
	}
	return output.RenderPoll(opts.IOStreams, info)
}

func newDo(f *runtime.Invocation) actionmessage.VoteFunc {
	return func(ctx context.Context, q actionmessage.VoteQuery) (output.PollInfo, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.PollInfo{}, err
		}
		var info output.PollInfo
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				info, err = telegram.VotePoll(ctx, api, res, q)
				return err
			})
		return info, err
	}
}
