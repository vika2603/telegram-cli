// Package ban implements "tg chat ban <ref> <user>" and "tg chat unban <ref> <user>".
package ban

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
	RawUser   string
	Unban     bool
	Yes       bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Prompter  ui.Prompter
	Ban       actionchat.BanFunc
}

// NewBan builds the cobra command for "tg chat ban".
func NewBan(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return banCmd(f, runF, false)
}

// NewUnban builds the cobra command for "tg chat unban".
func NewUnban(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return banCmd(f, runF, true)
}

func banCmd(f *runtime.Invocation, runF func(*Options) error, unban bool) *cobra.Command {
	opts := &Options{Unban: unban}
	use := "ban <ref> <user>"
	short := "Ban a user from a group or channel"
	if unban {
		use = "unban <ref> <user>"
		short = "Unban a user from a group or channel"
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.RawUser = args[1]
			opts.IOStreams = f.IOStreams
			opts.Prompter = f.Prompter
			if runF != nil {
				return runF(opts)
			}
			opts.Ban = newBanFn(f)
			return Run(cmd.Context(), opts)
		},
	}
	if !unban {
		cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"ref", "id", "kind", "title", "username"})
	return cmd
}

// Run executes the ban/unban logic.
func Run(ctx context.Context, opts *Options) error {
	pr, err := actionchat.Ban(ctx, actionchat.BanRequest{
		RawRef:   opts.RawRef,
		RawUser:  opts.RawUser,
		Unban:    opts.Unban,
		Yes:      opts.Yes,
		Prompter: opts.Prompter,
	}, opts.Ban)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, pr)
	}
	verb := "banned"
	if opts.Unban {
		verb = "unbanned"
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

func newBanFn(f *runtime.Invocation) actionchat.BanFunc {
	return func(ctx context.Context, q actionchat.BanQuery) (output.PeerRef, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.PeerRef{}, err
		}
		var pr output.PeerRef
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				pr, err = telegram.SetMemberBanned(ctx, api, res, q)
				return err
			})
		return pr, err
	}
}
