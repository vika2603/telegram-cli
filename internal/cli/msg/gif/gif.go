// Package gif implements "tg msg gif list".
package gif

import (
	"context"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// New builds the "tg msg gif" command group.
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gif",
		Short: "Your saved GIFs (for sending with `msg send --gif`)",
	}
	cmd.AddCommand(newList(f, nil))
	return cmd
}

// Options holds flags/deps for `gif list`.
type Options struct {
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionmessage.GifListFunc
}

func newList(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your saved GIFs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"kind", "ref", "id", "type"})
	return cmd
}

// Run dispatches and renders.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionmessage.ListGifs(ctx, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderStickers(opts.IOStreams, rows)
}

func newDo(f *runtime.Invocation) actionmessage.GifListFunc {
	return func(ctx context.Context) ([]output.StickerRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.StickerRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				rows, err = telegram.ListGifs(ctx, api)
				return err
			})
		return rows, err
	}
}
