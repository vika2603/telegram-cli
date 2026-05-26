// Package reply implements the top-level "tg reply" command.
package reply

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// Options holds resolved flags and injectable dependencies for Run.
type Options struct {
	RawMessageRef string
	Text          string
	Files         []string
	Names         []string
	Silent        bool
	Parse         string
	Exporter      output.Exporter
	IOStreams     *ui.IOStreams
	Stdin         io.Reader
	Send          actionmessage.SendFunc
}

// New builds the cobra command for "tg reply".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:               "reply <msg-ref> [text...]",
		Short:             "Reply to a message",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: complete.MessageRefs(f),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawMessageRef = args[0]
			opts.Text = strings.Join(args[1:], " ")
			opts.IOStreams = f.IOStreams
			opts.Stdin = f.IOStreams.In
			if runF != nil {
				return runF(opts)
			}
			opts.Send = newSend(f)
			return Run(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringArrayVar(&opts.Files, "file", nil, `File attachment; repeat for multiple files; "-" reads stdin bytes`)
	cmd.Flags().StringArrayVar(&opts.Names, "name", nil, "Upload filename override; repeat to match --file")
	cmd.Flags().BoolVar(&opts.Silent, "silent", false, "Send without notification")
	cmd.Flags().StringVar(&opts.Parse, "parse", "", "Parse mode for text or caption: html|markdown")
	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"action", "message_id", "chat_id", "date"})
	return cmd
}

// Run parses the target message ref, sends a reply, and renders the result.
func Run(ctx context.Context, opts *Options) error {
	msgRef, err := ref.ParseMessageRef(opts.RawMessageRef)
	if err != nil {
		return fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	rows, err := actionmessage.Send(ctx, actionmessage.SendRequest{
		RawRef:  msgRef.Peer.String(),
		Text:    opts.Text,
		Files:   opts.Files,
		Names:   opts.Names,
		ReplyTo: msgRef.MessageID,
		Silent:  opts.Silent,
		Parse:   opts.Parse,
		Stdin:   opts.Stdin,
	}, opts.Send)
	if err != nil {
		return err
	}
	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderSendResults(opts.IOStreams, rows)
}

func newSend(f *runtime.Invocation) actionmessage.SendFunc {
	return func(ctx context.Context, q actionmessage.SendQuery) ([]output.SendResultRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		// Daemon fast-path: text-only / no attachments. Stdin alone is
		// not a gate (CLI plumbs IOStreams.In into every Query, but
		// telegram.SendMessage only consumes it via Attachment.Path=="-").
		// Mirrors canDaemonSend in internal/cli/msg/send.
		if len(q.Attachments) == 0 {
			if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
				defer func() { _ = cl.Close() }()
				// Strip Stdin io.Reader before wire encode — JSON
				// cannot rehydrate the interface; see canDaemonSend
				// comment for full rationale.
				wire := q
				wire.Stdin = nil
				raw, err := cl.Call(ctx, "msg.send", wire)
				if err != nil {
					return nil, err
				}
				var rows []output.SendResultRow
				if err := json.Unmarshal(raw, &rows); err != nil {
					return nil, err
				}
				if store, err := account.OpenRecentStore(acct.Meta.Name); err == nil {
					recordSentMessages(store, q.Ref.String(), q.Text, rows)
				}
				return rows, nil
			}
		}

		var rows []output.SendResultRow
		err = f.WithPeers(ctx, acct, runtime.ClientOptsFrom(f, acct),
			func(ctx context.Context, api *tg.Client, _ *peers.Manager, res *peer.Resolver) error {
				rows, err = telegram.SendMessage(ctx, api, res, q, f.IOStreams.ErrOut)
				if err == nil {
					recordSentMessages(res.Store(), q.Ref.String(), q.Text, rows)
				}
				return err
			})
		return rows, err
	}
}

func recordSentMessages(store *account.PeerStore, peerRef, text string, rows []output.SendResultRow) {
	if store == nil {
		return
	}
	for _, row := range rows {
		if row.MessageID <= 0 {
			continue
		}
		_ = store.RecordRecentMessage(account.RecentMessage{
			Ref:       peerRef + ":" + strconv.Itoa(row.MessageID),
			PeerRef:   peerRef,
			MessageID: row.MessageID,
			Date:      row.Date,
			Text:      text,
		})
	}
}
