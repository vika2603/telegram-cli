// Package perms implements "tg chat perms" (default member permissions).
package perms

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

// Options holds flags/deps for perms.
type Options struct {
	RawRef    string
	Allow     []string
	Deny      []string
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionchat.PermsFunc
}

// New builds "tg chat perms".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "perms <ref>",
		Short:             "Set a group's default member permissions",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringSliceVar(&opts.Deny, "deny", nil, "Revoke for all members: send,media,stickers,bots,polls,links,invite,pin,info,topics")
	cmd.Flags().StringSliceVar(&opts.Allow, "allow", nil, "Grant for all members (same keywords)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "peer", "denied", "until"})
	return cmd
}

// Run validates and dispatches.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionchat.Perms(ctx, actionchat.PermsRequest{
		RawRef: opts.RawRef,
		Allow:  opts.Allow,
		Deny:   opts.Deny,
	}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderRights(opts.IOStreams, row)
}

func newDo(f *runtime.Invocation) actionchat.PermsFunc {
	return func(ctx context.Context, q actionchat.PermsQuery) (output.RightsRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.RightsRow{}, err
		}
		var row output.RightsRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.SetDefaultPerms(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
