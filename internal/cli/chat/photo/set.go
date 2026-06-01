package photo

import (
	"context"
	"fmt"
	"io"

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

// Options holds the resolved flags and injected dependencies for set/clear.
type Options struct {
	RawRef    string
	Path      string
	Clear     bool
	Stdin     io.Reader
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionchat.PhotoFunc
}

func newSet(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "set <ref> <path>",
		Short:             "Set a group/channel photo ('-' reads stdin bytes)",
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.Path = args[1]
			opts.IOStreams = f.IOStreams
			opts.Stdin = f.IOStreams.In
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f, opts.Stdin)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "peer"})
	return cmd
}

func newClear(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{Clear: true}
	cmd := &cobra.Command{
		Use:               "clear <ref>",
		Short:             "Remove a group/channel photo",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f, nil)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "peer"})
	return cmd
}

// Run validates and dispatches the set/clear operation.
func Run(ctx context.Context, opts *Options) error {
	req := actionchat.PhotoRequest{RawRef: opts.RawRef, Path: opts.Path}
	var (
		row output.ChatPhotoRow
		err error
	)
	if opts.Clear {
		row, err = actionchat.ClearPhoto(ctx, req, opts.Do)
	} else {
		row, err = actionchat.SetPhoto(ctx, req, opts.Do)
	}
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	verb := "set photo on"
	if opts.Clear {
		verb = "cleared photo on"
	}
	_, err = fmt.Fprintf(opts.IOStreams.Out, "%s %s\n", verb, peerName(row.Peer))
	return err
}

func peerName(p output.PeerRef) string {
	switch {
	case p.Username != "":
		return "@" + p.Username
	case p.Title != "":
		return p.Title
	default:
		return p.Ref
	}
}

func newDo(f *runtime.Invocation, stdin io.Reader) actionchat.PhotoFunc {
	return func(ctx context.Context, q actionchat.PhotoQuery) (output.ChatPhotoRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ChatPhotoRow{}, err
		}
		var row output.ChatPhotoRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.SetChatPhoto(ctx, api, res, q, stdin)
				return err
			})
		return row, err
	}
}
