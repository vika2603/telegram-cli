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

var inviteLinkJSONFields = []string{
	"action", "link", "title", "revoked", "permanent", "request_needed",
	"expire_date", "usage_limit", "usage", "requested", "admin_id", "created_at",
}

// CreateOptions holds the resolved flags and injected dependencies for create.
type CreateOptions struct {
	RawRef        string
	Title         string
	Expire        string
	UsageLimit    int
	RequestNeeded bool
	Exporter      output.Exporter
	IOStreams     *ui.IOStreams
	Do            actionchat.InviteLinkCreateFunc
}

func newCreate(f *runtime.Invocation, runF func(*CreateOptions) error) *cobra.Command {
	opts := &CreateOptions{}
	cmd := &cobra.Command{
		Use:               "create <ref>",
		Short:             "Create an invite link for a group or channel",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newCreateFn(f)
			return runCreate(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.Title, "title", "", "Label for the link")
	cmd.Flags().StringVar(&opts.Expire, "expire", "", "Expiry: RFC3339 timestamp or duration (e.g. 24h, 7d)")
	cmd.Flags().IntVar(&opts.UsageLimit, "usage-limit", 0, "Max number of joins (0 = unlimited)")
	cmd.Flags().BoolVar(&opts.RequestNeeded, "request-needed", false, "Joins require admin approval")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, inviteLinkJSONFields)
	return cmd
}

func runCreate(ctx context.Context, opts *CreateOptions) error {
	row, err := actionchat.InviteLinkCreate(ctx, actionchat.InviteLinkCreateRequest{
		RawRef:        opts.RawRef,
		Title:         opts.Title,
		Expire:        opts.Expire,
		UsageLimit:    opts.UsageLimit,
		RequestNeeded: opts.RequestNeeded,
	}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderInviteLink(opts.IOStreams, row)
}

func newCreateFn(f *runtime.Invocation) actionchat.InviteLinkCreateFunc {
	return func(ctx context.Context, q actionchat.InviteLinkCreateQuery) (output.InviteLinkRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.InviteLinkRow{}, err
		}
		var row output.InviteLinkRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.CreateInviteLink(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
