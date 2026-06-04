// Package admin implements "tg chat admin promote" and "tg chat admin demote".
package admin

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

// Options holds the resolved flags and injected dependencies for Run.
type Options struct {
	RawRef    string
	RawUser   string
	Demote    bool
	Title     string
	SetTitle  bool
	Rights    []string
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Promote   actionchat.PromoteFunc
}

// New builds the "tg chat admin" command group.
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Manage group/channel administrators",
	}
	cmd.AddCommand(NewPromote(f, nil))
	cmd.AddCommand(NewDemote(f, nil))
	return cmd
}

// NewPromote builds the cobra command for "tg chat admin promote".
func NewPromote(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return adminCmd(f, runF, false)
}

// NewDemote builds the cobra command for "tg chat admin demote".
func NewDemote(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return adminCmd(f, runF, true)
}

func adminCmd(f *runtime.Invocation, runF func(*Options) error, demote bool) *cobra.Command {
	opts := &Options{Demote: demote}
	use := "promote <ref> <user>"
	short := "Promote a user to admin in a group or channel"
	if demote {
		use = "demote <ref> <user>"
		short = "Remove admin rights from a user in a group or channel"
	}
	cmd := &cobra.Command{
		Use:               use,
		Short:             short,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.RawUser = args[1]
			opts.IOStreams = f.IOStreams
			opts.SetTitle = cmd.Flags().Changed("title")
			if runF != nil {
				return runF(opts)
			}
			opts.Promote = newPromoteFn(f)
			return Run(cmd.Context(), opts)
		},
	}
	if !demote {
		cmd.Flags().StringVar(&opts.Title, "title", "", "Custom admin title/rank (<=16 chars); omit to keep the current title, pass \"\" to clear it")
		cmd.Flags().StringSliceVar(&opts.Rights, "rights", nil, "Admin rights to grant: info,post,edit,delete,ban,invite,pin,add_admins,anonymous,call,topics,post_stories,edit_stories,delete_stories (default: broad set)")
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "peer", "granted"})
	return cmd
}

// Run executes the promote/demote logic.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionchat.Promote(ctx, actionchat.PromoteRequest{
		RawRef:   opts.RawRef,
		RawUser:  opts.RawUser,
		Demote:   opts.Demote,
		Title:    opts.Title,
		SetTitle: opts.SetTitle,
		Rights:   opts.Rights,
	}, opts.Promote)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderRights(opts.IOStreams, row)
}

func newPromoteFn(f *runtime.Invocation) actionchat.PromoteFunc {
	return func(ctx context.Context, q actionchat.PromoteQuery) (output.RightsRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.RightsRow{}, err
		}
		var row output.RightsRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.SetMemberAdmin(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
