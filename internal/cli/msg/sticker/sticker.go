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
	cmd.AddCommand(newFave(f, nil, false, "fave", "Add a sticker to favorites"))
	cmd.AddCommand(newFave(f, nil, true, "unfave", "Remove a sticker from favorites"))
	cmd.AddCommand(newSet(f, nil, false, "add", "Install a sticker set"))
	cmd.AddCommand(newSet(f, nil, true, "remove", "Uninstall a sticker set"))
	return cmd
}

// SetOptions holds flags/deps for `sticker add` / `remove`.
type SetOptions struct {
	RawSet    string
	Remove    bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionmessage.StickerSetFunc
}

func newSet(f *runtime.Invocation, runF func(*SetOptions) error, remove bool, use, short string) *cobra.Command {
	opts := &SetOptions{Remove: remove}
	cmd := &cobra.Command{
		Use:   use + " <set>",
		Short: short,
		Long:  short + ". <set> is a sticker set short name or an https://t.me/addstickers/<name> link.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawSet = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newSetDo(f)
			return RunSet(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "set", "title", "count", "archived"})
	return cmd
}

// RunSet dispatches install/uninstall.
func RunSet(ctx context.Context, opts *SetOptions) error {
	res, err := actionmessage.InstallStickerSet(ctx, actionmessage.StickerSetRequest{RawSet: opts.RawSet, Remove: opts.Remove}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, res)
	}
	return output.RenderStickerSet(opts.IOStreams, res)
}

func newSetDo(f *runtime.Invocation) actionmessage.StickerSetFunc {
	return func(ctx context.Context, q actionmessage.StickerSetQuery) (output.StickerSetResult, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.StickerSetResult{}, err
		}
		var res output.StickerSetResult
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, _ *peer.Resolver) error {
				res, err = telegram.InstallStickerSet(ctx, api, q)
				return err
			})
		return res, err
	}
}

// FaveOptions holds flags/deps for `sticker fave` / `unfave`.
type FaveOptions struct {
	RawRef    string
	Unfave    bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionmessage.FaveFunc
}

func newFave(f *runtime.Invocation, runF func(*FaveOptions) error, unfave bool, use, short string) *cobra.Command {
	opts := &FaveOptions{Unfave: unfave}
	cmd := &cobra.Command{
		Use:   use + " <ref>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newFaveDo(f)
			return RunFave(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "id"})
	return cmd
}

// RunFave dispatches fave/unfave.
func RunFave(ctx context.Context, opts *FaveOptions) error {
	res, err := actionmessage.FaveSticker(ctx, actionmessage.FaveRequest{RawRef: opts.RawRef, Unfave: opts.Unfave}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, res)
	}
	return output.RenderFave(opts.IOStreams, res)
}

func newFaveDo(f *runtime.Invocation) actionmessage.FaveFunc {
	return func(ctx context.Context, q actionmessage.FaveQuery) (output.FaveResult, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.FaveResult{}, err
		}
		var res output.FaveResult
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res2 *peer.Resolver) error {
				res, err = telegram.FaveSticker(ctx, api, res2, q)
				return err
			})
		return res, err
	}
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
