// Package edit implements "tg chat edit <ref>".
package edit

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
	Title     *string
	About     *string
	Username  *string
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Edit      actionchat.EditChatFunc
}

// New builds the cobra command for "tg chat edit". It is shared by
// "tg channel edit" via NewWith.
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return NewWith(f, runF, "Edit a supergroup's title and/or about text")
}

// NewWith builds the edit command with a caller-supplied short description so
// the channel tree can reuse it with channel-specific help.
func NewWith(f *runtime.Invocation, runF func(*Options) error, short string) *cobra.Command {
	opts := &Options{}
	var title, about, public string
	var private bool
	cmd := &cobra.Command{
		Use:               "edit <ref>",
		Short:             short,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if cmd.Flags().Changed("title") {
				opts.Title = &title
			}
			if cmd.Flags().Changed("about") {
				opts.About = &about
			}
			publicSet := cmd.Flags().Changed("public")
			if publicSet && private {
				return fmt.Errorf("%w: --public and --private are mutually exclusive", command.ErrUsage)
			}
			switch {
			case publicSet:
				opts.Username = &public
			case private:
				empty := ""
				opts.Username = &empty
			}
			if runF != nil {
				return runF(opts)
			}
			opts.Edit = newEditFn(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "New title")
	cmd.Flags().StringVar(&about, "about", "", "New description / about text")
	cmd.Flags().StringVar(&public, "public", "", "Make public with this @username")
	cmd.Flags().BoolVar(&private, "private", false, "Make private (remove the public username, invite-only)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"peer", "title", "type"})
	return cmd
}

// Run dispatches the edit request and renders the updated chat.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionchat.EditChat(ctx, actionchat.EditChatRequest{
		RawRef:   opts.RawRef,
		Title:    opts.Title,
		About:    opts.About,
		Username: opts.Username,
	}, opts.Edit)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderChatShow(opts.IOStreams, row)
}

func newEditFn(f *runtime.Invocation) actionchat.EditChatFunc {
	return func(ctx context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ChatRow{}, err
		}
		var row output.ChatRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.EditChat(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
