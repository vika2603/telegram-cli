package invite

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

// ListOptions holds the resolved flags and injected dependencies for list.
type ListOptions struct {
	RawRef    string
	RawAdmin  string
	Revoked   bool
	Limit     int
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Fetch     actionchat.InviteLinkListFunc
}

func newList(f *runtime.Invocation, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{}
	cmd := &cobra.Command{
		Use:               "list <ref>",
		Short:             "List invite links for a group or channel",
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
	cmd.Flags().BoolVar(&opts.Revoked, "revoked", false, "List revoked links instead of active ones")
	cmd.Flags().StringVar(&opts.RawAdmin, "admin", "", "List links created by this admin (default: yourself)")
	cmd.Flags().IntVar(&opts.Limit, "limit", 100, "Max links to list")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, inviteLinkJSONFields)
	return cmd
}

func runList(ctx context.Context, opts *ListOptions) error {
	rows, err := actionchat.InviteLinkList(ctx, actionchat.InviteLinkListRequest{
		RawRef:   opts.RawRef,
		RawAdmin: opts.RawAdmin,
		Revoked:  opts.Revoked,
		Limit:    opts.Limit,
	}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderInviteLinkList(opts.IOStreams, rows)
}

func newListFn(f *runtime.Invocation) actionchat.InviteLinkListFunc {
	return func(ctx context.Context, q actionchat.InviteLinkListQuery) ([]output.InviteLinkRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.InviteLinkRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.ListInviteLinks(ctx, api, res, q)
				return err
			})
		return rows, err
	}
}
