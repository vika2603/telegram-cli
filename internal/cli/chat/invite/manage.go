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

// ManageOptions holds flags/deps for `invite revoke` and `invite delete`.
type ManageOptions struct {
	RawRef    string
	Link      string
	Delete    bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionchat.InviteLinkFunc
}

func newRevoke(f *runtime.Invocation, runF func(*ManageOptions) error) *cobra.Command {
	return manageCmd(f, runF, false, "revoke", "Revoke an invite link")
}

func newDelete(f *runtime.Invocation, runF func(*ManageOptions) error) *cobra.Command {
	return manageCmd(f, runF, true, "delete", "Delete a revoked invite link")
}

func manageCmd(f *runtime.Invocation, runF func(*ManageOptions) error, del bool, use, short string) *cobra.Command {
	opts := &ManageOptions{Delete: del}
	cmd := &cobra.Command{
		Use:               use + " <ref> <link>",
		Short:             short,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.Link = args[1]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newManageFn(f, del)
			return runManage(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, inviteLinkJSONFields)
	return cmd
}

func runManage(ctx context.Context, opts *ManageOptions) error {
	req := actionchat.InviteLinkRequest{RawRef: opts.RawRef, Link: opts.Link}
	var (
		row output.InviteLinkRow
		err error
	)
	if opts.Delete {
		row, err = actionchat.InviteLinkDelete(ctx, req, opts.Do)
	} else {
		row, err = actionchat.InviteLinkRevoke(ctx, req, opts.Do)
	}
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderInviteLink(opts.IOStreams, row)
}

func newManageFn(f *runtime.Invocation, del bool) actionchat.InviteLinkFunc {
	return func(ctx context.Context, q actionchat.InviteLinkQuery) (output.InviteLinkRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.InviteLinkRow{}, err
		}
		var row output.InviteLinkRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				if del {
					row, err = telegram.DeleteInviteLink(ctx, api, res, q)
				} else {
					row, err = telegram.RevokeInviteLink(ctx, api, res, q)
				}
				return err
			})
		return row, err
	}
}
