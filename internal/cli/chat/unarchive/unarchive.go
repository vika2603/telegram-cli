// Package unarchive implements "tg chat unarchive <ref>".
package unarchive

import (
	"context"
	"encoding/json"

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

// Options holds flag values and injected dependencies for Run.
type Options struct {
	RawRef    string
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionchat.FolderFunc
}

// New builds the cobra command for "tg chat unarchive".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "unarchive <ref>",
		Short: "Move a chat back to the main folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
			opts.IOStreams = f.IOStreams
			if runF != nil {
				return runF(opts)
			}
			opts.Do = newDo(f)
			return Run(cmd.Context(), opts)
		},
	}
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "peer", "folder"})
	return cmd
}

// Run validates and dispatches.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionchat.Unarchive(ctx, actionchat.FolderRequest{RawRef: opts.RawRef}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.WriteChatFolderJSON(opts.IOStreams.Out, row)
}

func newDo(f *runtime.Invocation) actionchat.FolderFunc {
	return func(ctx context.Context, q actionchat.FolderQuery) (output.ChatFolderRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ChatFolderRow{}, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "chat.folder", q)
			if err != nil {
				return output.ChatFolderRow{}, err
			}
			var row output.ChatFolderRow
			if err := json.Unmarshal(raw, &row); err != nil {
				return output.ChatFolderRow{}, err
			}
			return row, nil
		}

		var row output.ChatFolderRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.MoveChatToFolder(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
