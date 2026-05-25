// Package join implements "tg chat join <ref>".
package join

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
	Do        actionchat.MembershipFunc
}

// New builds the cobra command for "tg chat join".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "join <ref>",
		Short: "Join a chat, channel, or invite link",
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
	output.AddJSONFlags(cmd, &opts.Exporter, []string{"action", "peer", "already_member", "role"})
	return cmd
}

// Run validates and dispatches.
func Run(ctx context.Context, opts *Options) error {
	row, err := actionchat.Join(ctx, actionchat.MembershipRequest{RawRef: opts.RawRef}, opts.Do)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, row)
	}
	return output.WriteChatMembershipJSON(opts.IOStreams.Out, row)
}

func newDo(f *runtime.Invocation) actionchat.MembershipFunc {
	return func(ctx context.Context, q actionchat.MembershipQuery) (output.ChatMembershipRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return output.ChatMembershipRow{}, err
		}
		if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
			defer func() { _ = cl.Close() }()
			raw, err := cl.Call(ctx, "chat.join", q)
			if err != nil {
				return output.ChatMembershipRow{}, err
			}
			var row output.ChatMembershipRow
			if err := json.Unmarshal(raw, &row); err != nil {
				return output.ChatMembershipRow{}, err
			}
			return row, nil
		}

		var row output.ChatMembershipRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				row, err = telegram.JoinChat(ctx, api, res, q)
				return err
			})
		return row, err
	}
}
