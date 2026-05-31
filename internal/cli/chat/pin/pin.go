// Package pin implements "tg chat pin <ref>" and "tg chat unpin <ref>".
package pin

import (
	"context"
	"encoding/json"

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

// Options holds flag values and injected dependencies for Run.
type Options struct {
	RawRef    string
	Unpin     bool
	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Do        actionchat.PinFunc
}

// NewPin builds "tg chat pin"; NewUnpin builds the "unpin" sibling.
func NewPin(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return newCmd(f, runF, false, "pin", "Pin a chat to the top of the chat list")
}

// NewUnpin builds "tg chat unpin".
func NewUnpin(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	return newCmd(f, runF, true, "unpin", "Unpin a chat from the top of the chat list")
}

func newCmd(f *runtime.Invocation, runF func(*Options) error, unpin bool, use, short string) *cobra.Command {
	opts := &Options{Unpin: unpin}
	cmd := &cobra.Command{
		Use:               use + " <ref>",
		Short:             short,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
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
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "peer", "pinned"})
	return cmd
}

// Run validates and dispatches.
func Run(ctx context.Context, opts *Options) error {
	req := actionchat.PinRequest{RawRef: opts.RawRef}
	var (
		row output.ChatPinRow
		err error
	)
	if opts.Unpin {
		row, err = actionchat.Unpin(ctx, req, opts.Do)
	} else {
		row, err = actionchat.Pin(ctx, req, opts.Do)
	}
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.WriteChatPinJSON(opts.IOStreams.Out, row)
}

func newDo(f *runtime.Invocation) actionchat.PinFunc {
	return func(ctx context.Context, q actionchat.PinQuery) (output.ChatPinRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ChatPinRow{}, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "chat.pin", q)
			if err != nil {
				return output.ChatPinRow{}, err
			}
			var row output.ChatPinRow
			if err := json.Unmarshal(raw, &row); err != nil {
				return output.ChatPinRow{}, err
			}
			return row, nil
		}

		var row output.ChatPinRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.PinChat(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
