package join

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

// ListOptions holds flags/deps for `join list`.
type ListOptions struct {
	RawRef    string
	Link      string
	Limit     int
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Fetch     actionchat.JoinListFunc
}

func newList(f *runtime.Invocation, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{}
	cmd := &cobra.Command{
		Use:               "list <ref>",
		Short:             "List pending join requests",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newListFn(f)
			return runList(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Link, "link", "", "Only requests that came via this invite link")
	cmd.Flags().IntVar(&opts.Limit, "limit", 100, "Max requests to list")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"user_id", "username", "first_name", "last_name", "is_bot", "role", "joined_at"})
	return cmd
}

func runList(ctx context.Context, opts *ListOptions) error {
	rows, err := actionchat.JoinList(ctx, actionchat.JoinListRequest{
		RawRef: opts.RawRef,
		Link:   opts.Link,
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

func newListFn(f *runtime.Invocation) actionchat.JoinListFunc {
	return func(ctx context.Context, q actionchat.JoinListQuery) ([]output.MemberRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.MemberRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.ListJoinRequests(ctx, api, res, q)
				return err
			})
		return rows, err
	}
}
