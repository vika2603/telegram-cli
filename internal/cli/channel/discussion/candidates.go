package discussion

import (
	"context"

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

// CandidatesOptions holds the injected dependencies for the candidates run.
type CandidatesOptions struct {
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Fetch     actionchat.DiscussionCandidatesFunc
}

func newCandidates(f *runtime.Invocation, runF func(*CandidatesOptions) error) *cobra.Command {
	opts := &CandidatesOptions{}
	cmd := &cobra.Command{
		Use:   "candidates",
		Short: "List supergroups eligible to be a channel's discussion group",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch(f)
			return runCandidates(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"peer", "title", "type"})
	return cmd
}

func runCandidates(ctx context.Context, opts *CandidatesOptions) error {
	rows, err := actionchat.DiscussionCandidates(ctx, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderChatList(opts.IOStreams, rows)
}

func newFetch(f *runtime.Invocation) actionchat.DiscussionCandidatesFunc {
	return func(ctx context.Context) ([]output.ChatRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.ChatRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				rows, err = telegram.ListDiscussionCandidates(ctx, api)
				return err
			})
		return rows, err
	}
}
