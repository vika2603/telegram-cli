// Package members implements "tg chat members <ref>".
package members

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

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawRef    string
	Filter    string
	Q         string
	Limit     int
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Fetch     actionchat.MembersFunc
}

// New builds the cobra command for "tg chat members".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "members <ref>",
		Short: "List members of a group or channel (not available yet)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Filter, "filter", "recent",
		"Filter: recent|admins|bots|kicked|banned|contacts")
	cmd.Flags().StringVar(&opts.Q, "q", "",
		"Substring filter (valid only with --filter kicked|banned|contacts)")
	cmd.Flags().IntVar(&opts.Limit, "limit", 30, "Max members (cap 1000)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"user_id", "username", "first_name", "last_name", "is_bot", "role", "joined_at"})
	return cmd
}

// Run executes the members logic using opts.Fetch for data retrieval.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionchat.Members(ctx, actionchat.MembersRequest{
		RawRef: opts.RawRef,
		Filter: opts.Filter,
		Q:      opts.Q,
		Limit:  opts.Limit,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderMembers(opts.IOStreams, rows)
}

func newFetch(f *runtime.Invocation) actionchat.MembersFunc {
	return func(ctx context.Context, q actionchat.MembersQuery) ([]output.MemberRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.MemberRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, _ *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.ListChatMembers(ctx, res, q)
				return err
			})
		return rows, err
	}
}
