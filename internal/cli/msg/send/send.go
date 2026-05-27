// Package send implements "tg msg send <ref> [text...]".
package send

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/spf13/cobra"

	"github.com/vika2603/telegram-cli/internal/account"
	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
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
	RawRef   string
	Text     string
	Files    []string
	Names    []string
	ReplyTo  int
	Silent   bool
	Schedule time.Time
	Parse    string

	Exporter  output.Exporter
	IOStreams *ui.IOStreams
	Stdin     io.Reader

	// Send is the closure that performs the actual Telegram call. Production
	// code sets it via newSend; tests stub it directly.
	Send actionmessage.SendFunc
}

// New builds the cobra command for "tg msg send".
func New(f *runtime.Invocation, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	var scheduleRaw string

	cmd := &cobra.Command{
		Use:               "send <ref> [text...]",
		Short:             "Send text and optional media to a chat",
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: complete.PeerRefs(f),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if scheduleRaw != "" {
				t, err := time.Parse(time.RFC3339, scheduleRaw)
				if err != nil {
					return fmt.Errorf("%w: --schedule must be RFC3339: %s", command.ErrUsage, err.Error())
				}
				opts.Schedule = t
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RawRef = args[0]
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
	cmd.Flags().IntVar(&opts.ReplyTo, "reply-to", 0, "Reply to message ID")
	cmd.Flags().BoolVar(&opts.Silent, "silent", false, "Send without notification")
	cmd.Flags().StringVar(&scheduleRaw, "schedule", "", "Schedule delivery (RFC3339)")
	cmd.Flags().StringVar(&opts.Parse, "parse", "", "Parse mode for text or caption (only: html)")

	command.SetMeta(cmd, command.Meta{NeedsAccount: true, NeedsClient: true})
	output.AddJSONFlags(cmd, &opts.Exporter,
		[]string{"action", "message_id", "chat_id", "date"})
	return cmd
}

// Run dispatches the normalized request and renders the result.
func Run(ctx context.Context, opts *Options) error {
	rows, err := actionmessage.Send(ctx, actionmessage.SendRequest{
		RawRef:   opts.RawRef,
		Text:     opts.Text,
		Files:    opts.Files,
		Names:    opts.Names,
		ReplyTo:  opts.ReplyTo,
		Silent:   opts.Silent,
		Schedule: opts.Schedule,
		Parse:    opts.Parse,
		Stdin:    opts.Stdin,
	}, opts.Send)
	if err != nil {
		return err
	}

	if opts.Exporter != nil {
		return opts.Exporter.Write(opts.IOStreams, rows)
	}
	return output.RenderSendResults(opts.IOStreams, rows)
}

// newSend returns the production Send closure that calls the Telegram API.
//
// Daemon fast-path applies only when the payload is pure text — file
// attachments and stdin require byte streams the IPC socket does not
// carry yet, so those fall through to the local WithPeers path.
func newSend(f *runtime.Invocation) actionmessage.SendFunc {
	return func(ctx context.Context, q actionmessage.SendQuery) ([]output.SendResultRow, error) {
		acct, err := f.Account("")
		if err != nil {
			return nil, err
		}
		if canDaemonSend(q) {
			if cl, _ := runtime.MaybeDialDaemon(ctx, f, acct); cl != nil {
				defer func() { _ = cl.Close() }()
				// SendQuery.Stdin is io.Reader, which the JSON encoder
				// renders as `{}` and the decoder cannot rehydrate.
				// Strip it before sending — telegram.SendMessage only
				// consumes Stdin via Attachment.Path == "-", which the
				// canDaemonSend gate already excluded.
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
				rows, err = telegram.SendMessage(ctx, api, res, q)
				if err == nil {
					recordSentMessages(res.Store(), q.Ref.String(), q.Text, rows)
				}
				return err
			})
		return rows, err
	}
}

// canDaemonSend reports whether a SendQuery is daemon-routable.
// Attachments require bytes the daemon cannot read from the client's
// filesystem, so file uploads fall back to local mode. Stdin alone
// is NOT a gate — the CLI plumbs IOStreams.In into every SendQuery
// for use during stdin-text or stdin-file paths, but telegram.SendMessage
// only consumes Stdin when an Attachment with Path == "-" is present,
// and Normalize already resolves stdin-as-text into Text before this
// point. Schedule + Silent + Parse + ReplyTo are pure metadata and
// stay supported.
func canDaemonSend(q actionmessage.SendQuery) bool {
	return len(q.Attachments) == 0
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
