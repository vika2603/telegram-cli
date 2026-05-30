// Package create implements "tg channel create <title>".
package create

import (
	"context"

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
	Title     string
	About     string
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Create    actionchat.CreateChatFunc
}

// New builds the cobra command for "tg channel create".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "create <title>",
		Short: "Create a broadcast channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Title = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Create = newCreate(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.About, "about", "", "Description / about text")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"peer", "title", "type"})
	return cmd
}

// Run dispatches the create request and renders the new channel.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionchat.CreateChat(ctx, actionchat.CreateChatRequest{
		Title:     opts.Title,
		About:     opts.About,
		Broadcast: true,
	}, opts.Create)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderChatShow(opts.IOStreams, row)
}

func newCreate(f *runtime.Invocation) actionchat.CreateChatFunc {
	return func(ctx context.Context, q actionchat.CreateChatQuery) (output.ChatRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ChatRow{}, err
		}
		var row output.ChatRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				row, err = telegram.CreateChat(ctx, api, q)
				return err
			})
		return row, err
	}
}
