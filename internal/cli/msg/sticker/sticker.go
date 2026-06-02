// Package sticker implements "tg msg sticker list".
package sticker

import (
	"context"
	"fmt"

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

// New builds the "tg msg sticker" command group.
func New(f *runtime.Invocation) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sticker",
		Short: "Browse your stickers (for sending with `msg send --sticker`)",
	}
	cmd.AddCommand(newList(f, nil))
	return cmd
}

// Options holds flags/deps for `sticker list`.
type Options struct {
	Recent    bool
	Faved     bool
	Installed bool
	All       bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionmessage.StickerListFunc
}

func newList(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent/favorited stickers or installed sets",
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
	cmd.Flags().BoolVar(&opts.Recent, "recent", false, "Recently used stickers (default)")
	cmd.Flags().BoolVar(&opts.Faved, "faved", false, "Favorited stickers")
	cmd.Flags().BoolVar(&opts.Installed, "installed", false, "Installed sticker sets")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Every sticker: recent + faved + all installed sets expanded (slow)")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"kind", "ref", "id", "emoji", "type", "set", "title", "count"})
	return cmd
}

// Run resolves the source flags and dispatches.
func Run(ctx context.Context, opts *Options) error {
	source, err := opts.source()
	if err != nil {
		return err
	}
	rows, err := actionmessage.ListStickers(ctx, actionmessage.StickerListRequest{Source: source}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderStickers(opts.IOStreams, rows)
}

// source maps the mutually-exclusive flags to a single source (default recent).
func (o *Options) source() (actionmessage.StickerListSource, error) {
	n := 0
	var src actionmessage.StickerListSource
	for _, c := range []struct {
		on  bool
		val actionmessage.StickerListSource
	}{
		{o.Recent, actionmessage.StickerRecent},
		{o.Faved, actionmessage.StickerFaved},
		{o.Installed, actionmessage.StickerInstalled},
		{o.All, actionmessage.StickerAll},
	} {
		if c.on {
			n++
			src = c.val
		}
	}
	if n > 1 {
		return "", fmt.Errorf("%w: choose only one of --recent/--faved/--installed/--all", command.ErrUsage)
	}
	if n == 0 {
		return actionmessage.StickerRecent, nil
	}
	return src, nil
}

func newDo(f *runtime.Invocation) actionmessage.StickerListFunc {
	return func(ctx context.Context, q actionmessage.StickerListQuery) ([]output.StickerRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		var rows []output.StickerRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				rows, err = telegram.ListStickers(ctx, api, q)
				return err
			})
		return rows, err
	}
}
