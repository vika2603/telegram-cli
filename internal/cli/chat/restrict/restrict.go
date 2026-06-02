// Package restrict implements "tg chat restrict" and "tg chat unrestrict".
package restrict

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

// Options holds flags/deps for restrict / unrestrict.
type Options struct {
	RawRef     string
	RawUser    string
	Allow      []string
	Deny       []string
	Until      string
	Unrestrict bool
	Exporter   output.Exporter
	IOStreams  *ui.IOStreams
	Do         actionchat.RestrictFunc
}

// NewRestrict builds "tg chat restrict".
func NewRestrict(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return restrictCmd(f, runF, false, "restrict", "Restrict a member's permissions (optionally for a duration)")
}

// NewUnrestrict builds "tg chat unrestrict".
func NewUnrestrict(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return restrictCmd(f, runF, true, "unrestrict", "Lift all restrictions on a member")
}

func restrictCmd(f *runtime.Invocation, runF func(*Options) error, unrestrict bool, use, short string) *cobra.Command {
	opts := &Options{Unrestrict: unrestrict}
	cmd := &cobra.Command{
		Use:               use + " <ref> <user>",
		Short:             short,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.RawUser = args[1]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f)
			return Run(cmd.Context(), opts)
		},
	}
	if !unrestrict {
		cmd.Flags().StringSliceVar(&opts.Deny, "deny", nil, "Revoke permissions: send,media,stickers,bots,polls,links,invite,pin,info,topics")
		cmd.Flags().StringSliceVar(&opts.Allow, "allow", nil, "Grant permissions (same keywords)")
		cmd.Flags().StringVar(&opts.Until, "until", "", "Restriction expiry: RFC3339 or duration (e.g. 1h, 7d); empty = permanent")
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "peer", "denied", "until"})
	return cmd
}

// Run validates and dispatches.
func Run(ctx context.Context, opts *Options) error {
	req := actionchat.RestrictRequest{
		RawRef:  opts.RawRef,
		RawUser: opts.RawUser,
		Allow:   opts.Allow,
		Deny:    opts.Deny,
		Until:   opts.Until,
	}
	var (
		row output.RightsRow
		err error
	)
	if opts.Unrestrict {
		row, err = actionchat.Unrestrict(ctx, req, opts.Do)
	} else {
		row, err = actionchat.Restrict(ctx, req, opts.Do)
	}
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderRights(opts.IOStreams, row)
}

func newDo(f *runtime.Invocation) actionchat.RestrictFunc {
	return func(ctx context.Context, q actionchat.RestrictQuery) (output.RightsRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.RightsRow{}, err
		}
		var row output.RightsRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.RestrictMember(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
