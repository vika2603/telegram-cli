// Package invite implements "tg chat invite <ref> <user>...".
package invite

import (
	"context"
	"fmt"

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
	RawUsers  []string
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Invite    actionchat.InviteFunc
}

// New builds the cobra command for "tg chat invite".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "invite <ref> <user>...",
		Short: "Add one or more users to a group or channel",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.RawUsers = args[1:]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Invite = newInviteFn(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"ref", "id", "kind", "title", "username"})
	return cmd
}

// Run executes the invite logic.
func Run(ctx context.Context, opts *Options) error {
	refs, err := actionchat.Invite(ctx, actionchat.InviteRequest{
		RawRef:   opts.RawRef,
		RawUsers: opts.RawUsers,
	}, opts.Invite)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, refs)
	}
	for _, r := range refs {
		name := r.Title
		if r.Username != "" {
			name = "@" + r.Username
		}
		if name == "" {
			name = r.Ref
		}
		if _, err := fmt.Fprintf(opts.IOStreams.Out, "invited %s\n", name); err != nil {
			return err
		}
	}
	return nil
}

func newInviteFn(f *runtime.Invocation) actionchat.InviteFunc {
	return func(ctx context.Context, q actionchat.InviteQuery) ([]output.PeerRef, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var refs []output.PeerRef
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				refs, err = telegram.InviteToChat(ctx, api, res, q)
				return err
			})
		return refs, err
	}
}
