// Package mute implements "tg chat mute <ref>".
package mute

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
	RawRef   string
	Duration string
	Until    string
	Forever  bool

	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionchat.MuteFunc
}

// New builds the cobra command for "tg chat mute".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "mute <ref>",
		Short: "Mute notifications from a chat",
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
	cmd.Flags().StringVar(&opts.Duration, "duration", "", "Mute for a duration (e.g. 1h, 3d). Mutually exclusive with --until / --forever.")
	cmd.Flags().StringVar(&opts.Until, "until", "", "Mute until RFC3339 timestamp. Mutually exclusive with --duration / --forever.")
	cmd.Flags().BoolVar(&opts.Forever, "forever", false, "Mute indefinitely (default when no flag given).")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "peer", "mute_until"})
	return cmd
}

// Run validates and dispatches.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionchat.Mute(ctx, actionchat.MuteRequest{
		RawRef:   opts.RawRef,
		Duration: opts.Duration,
		Until:    opts.Until,
		Forever:  opts.Forever,
	}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.WriteChatMuteJSON(opts.IOStreams.Out, row)
}

func newDo(f *runtime.Invocation) actionchat.MuteFunc {
	return func(ctx context.Context, q actionchat.MuteQuery) (output.ChatMuteRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ChatMuteRow{}, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "chat.mute", q)
			if err != nil {
				return output.ChatMuteRow{}, err
			}
			var row output.ChatMuteRow
			if err := json.Unmarshal(raw, &row); err != nil {
				return output.ChatMuteRow{}, err
			}
			return row, nil
		}

		var row output.ChatMuteRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.MuteChat(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
