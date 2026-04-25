// Package show implements "tg chat info <ref>".
package show

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
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Fetch     actionchat.ShowFunc
}

// New builds the cobra command for "tg chat info".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "info <ref>",
		Short:             "Show information about a chat, user, or channel",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Fetch = newFetch(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"peer", "title", "type", "unread", "pinned", "archived", "muted", "top_message", "last"})
	return cmd
}

// Run executes the show logic using opts.Fetch for data retrieval.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionchat.Show(ctx, actionchat.ShowRequest{RawRef: opts.RawRef}, opts.Fetch)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.RenderChatShow(opts.IOStreams, row)
}

func newFetch(f *runtime.Invocation) actionchat.ShowFunc {
	return func(ctx context.Context, q actionchat.ShowQuery) (output.ChatRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ChatRow{}, err
		}
		var row output.ChatRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, _ *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.ShowChat(ctx, res, q)
				return err
			})
		return row, err
	}
}
