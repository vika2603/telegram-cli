// Package admin implements "tg chat admin promote" and "tg chat admin demote".
package admin

import (
	"context"
	"fmt"

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
			if runF != nil {
				return runF(opts)
			}
			opts.Promote = newPromoteFn(f)
			return Run(cmd.Context(), opts)
		},
	}
	if !demote {
		cmd.Flags().StringVar(&opts.Title, "title", "", "Custom admin title/rank (<=16 chars)")
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"ref", "id", "kind", "title", "username"})
	return cmd
}

// Run executes the promote/demote logic.
func Run(ctx context.Context, opts *Options) error {
	pr, err := actionchat.Promote(ctx, actionchat.PromoteRequest{
		RawRef:  opts.RawRef,
		RawUser: opts.RawUser,
		Demote:  opts.Demote,
		Title:   opts.Title,
	}, opts.Promote)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, pr)
	}
	verb := "promoted"
	if opts.Demote {
		verb = "demoted"
	}
	name := pr.Title
	if pr.Username != "" {
		name = "@" + pr.Username
	}
	if name == "" {
		name = pr.Ref
	}
	_, err = fmt.Fprintf(opts.IOStreams.Out, "%s %s\n", verb, name)
	return err
}

func newPromoteFn(f *runtime.Invocation) actionchat.PromoteFunc {
	return func(ctx context.Context, q actionchat.PromoteQuery) (output.PeerRef, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.PeerRef{}, err
		}
		var pr output.PeerRef
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				pr, err = telegram.SetMemberAdmin(ctx, api, res, q)
				return err
			})
		return pr, err
	}
}
