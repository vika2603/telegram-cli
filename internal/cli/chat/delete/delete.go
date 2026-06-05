// Package deletecmd implements "tg chat delete <ref>".
package deletecmd

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
	Revoke    bool
	Yes       bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Prompter  ui.Prompter
	Delete    actionchat.DeleteChatFunc
}

// New builds the cobra command for "tg chat delete".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return NewWith(f, runF, "Delete a supergroup/channel, or remove a user DM from your chat list")
}

// NewWith builds the delete command with a caller-supplied short description so
// the "tg channel delete" tree can reuse this implementation (DeleteChat
// detects the peer type itself).
func NewWith(f *runtime.Invocation, runF func(*Options) error, short string) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "delete <ref>",
		Short:             short,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			opts.Prompter = f.Prompter
			if runF != nil {
				return runF(opts)
			}
			opts.Delete = newDeleteFn(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Revoke, "revoke", false, "For a user DM, also delete the conversation on the other side")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"id", "ref", "kind", "title", "username"})
	return cmd
}

// Run dispatches the delete request and renders the result.
func Run(ctx context.Context, opts *Options) error {
	pr, err := actionchat.DeleteChat(ctx, actionchat.DeleteChatRequest{
		RawRef:   opts.RawRef,
		Revoke:   opts.Revoke,
		Yes:      opts.Yes,
		Prompter: opts.Prompter,
	}, opts.Delete)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, pr)
	}
	_, err = fmt.Fprintf(opts.IOStreams.Out, "deleted %s\n", opts.RawRef)
	return err
}

func newDeleteFn(f *runtime.Invocation) actionchat.DeleteChatFunc {
	return func(ctx context.Context, q actionchat.DeleteChatQuery) (output.PeerRef, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.PeerRef{}, err
		}
		var pr output.PeerRef
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				pr, err = telegram.DeleteChat(ctx, api, res, q)
				return err
			})
		return pr, err
	}
}
