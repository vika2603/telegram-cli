package telegram

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gotd/td/telegram/downloader"
	gotdmessage "github.com/gotd/td/telegram/message"
	"github.com/gotd/td/telegram/message/html"
	"github.com/gotd/td/telegram/message/styling"
	"github.com/gotd/td/telegram/message/unpack"
	"github.com/gotd/td/telegram/query"
	msgquery "github.com/gotd/td/telegram/query/messages"
	"github.com/gotd/td/tg"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	telegrammessage "github.com/vika2603/telegram-cli/internal/telegram/message"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// SendMessage performs the Telegram RPC for `tg msg send`.
func SendMessage(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.SendQuery, errOut io.Writer) ([]output.SendResultRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}
	sender := gotdmessage.NewSender(api)
	b := sender.To(resolved.InputPeer)
	if q.Silent {
		b.Silent()
	}
	if q.ReplyTo > 0 {
		b.Reply(q.ReplyTo)
	}
	if !q.Schedule.IsZero() {
		b.Schedule(q.Schedule)
	}

	var upd tg.UpdatesClass
	if len(q.Attachments) > 0 {
		upd, err = sendAttachments(ctx, b, q, errOut)
	} else {
		switch q.Parse {
		case "html":
			upd, err = b.StyledText(ctx, html.String(nil, q.Text))
		case "markdown":
			_, _ = fmt.Fprintln(errOut, "--parse not supported by current gotd build; sending as plain text")
			upd, err = b.Text(ctx, q.Text)
		default:
			upd, err = b.Text(ctx, q.Text)
		}
	}
	if err != nil {
		return nil, err
	}

	return sentMessageRows("send", upd), nil
}

func sendAttachments(
	ctx context.Context,
	b *gotdmessage.RequestBuilder,
	q actionmessage.SendQuery,
	errOut io.Writer,
) (tg.UpdatesClass, error) {
	docs := make([]*gotdmessage.UploadedDocumentBuilder, 0, len(q.Attachments))
	for i, attachment := range q.Attachments {
		input, err := uploadAttachment(ctx, b, attachment, q.Stdin)
		if err != nil {
			return nil, err
		}
		var caption []styling.StyledTextOption
		if i == 0 {
			caption = captionStyling(q.Text, q.Parse, errOut)
		}
		doc := gotdmessage.File(input, caption...)
		if attachment.Name != "" {
			doc.Filename(attachment.Name)
		}
		docs = append(docs, doc)
	}
	if len(docs) == 1 {
		return b.Media(ctx, docs[0])
	}
	opts := make([]gotdmessage.MultiMediaOption, len(docs))
	for i, doc := range docs {
		opts[i] = doc
	}
	return b.Album(ctx, opts[0], opts[1:]...)
}

func uploadAttachment(
	ctx context.Context,
	b *gotdmessage.RequestBuilder,
	attachment actionmessage.Attachment,
	stdin io.Reader,
) (tg.InputFileClass, error) {
	if attachment.Path == "-" {
		if stdin == nil {
			return nil, fmt.Errorf("%w: --file - requires stdin to be available", command.ErrUsage)
		}
		return b.Upload(gotdmessage.FromReader(attachment.Name, stdin)).AsInputFile(ctx)
	}
	return b.Upload(gotdmessage.FromPath(attachment.Path)).AsInputFile(ctx)
}

// EditMessage performs the Telegram RPC for `tg msg edit`.
func EditMessage(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.EditQuery, errOut io.Writer) (output.SendResultRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.SendResultRow{}, err
	}
	eb := gotdmessage.NewSender(api).To(resolved.InputPeer).Edit(q.MessageID)
	switch q.Parse {
	case "html":
		_, err = eb.StyledText(ctx, html.String(nil, q.Text))
	case "markdown":
		_, _ = fmt.Fprintln(errOut, "--parse not supported by current gotd build; sending as plain text")
		_, err = eb.Text(ctx, q.Text)
	default:
		_, err = eb.Text(ctx, q.Text)
	}
	if err != nil {
		return output.SendResultRow{}, err
	}
	return output.SendResultRow{Action: "edit", MessageID: q.MessageID, ChatID: resolved.ID}, nil
}

// ForwardMessages performs the Telegram RPC for `tg msg forward`.
//
// Returns the dest message id (from the Updates the server sent back),
// NOT the source id. Multi-id forwards collapse to the first new
// message — callers needing every dest id should switch to a slice
// return; that's a follow-up. Falls back to the source id only if the
// server response carries no usable update, which shouldn't happen in
// practice but keeps the row non-empty so scripts have something.
func ForwardMessages(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.ForwardQuery) (output.SendResultRow, error) {
	from, err := resolver.Resolve(ctx, q.From)
	if err != nil {
		return output.SendResultRow{}, err
	}
	to, err := resolver.Resolve(ctx, q.To)
	if err != nil {
		return output.SendResultRow{}, err
	}
	b := gotdmessage.NewSender(api).To(to.InputPeer)
	if q.Silent {
		b.Silent()
	}
	upd, err := b.ForwardIDs(from.InputPeer, q.IDs[0], q.IDs[1:]...).Send(ctx)
	if err != nil {
		return output.SendResultRow{}, err
	}
	rows := sentMessageRows("forward", upd)
	if len(rows) == 0 {
		return output.SendResultRow{Action: "forward", MessageID: q.IDs[0], ChatID: to.ID}, nil
	}
	return rows[0], nil
}

// DeleteMessages performs the Telegram RPC for `tg msg delete`.
func DeleteMessages(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.DeleteQuery) error {
	sender := gotdmessage.NewSender(api)
	if !q.Revoke {
		_, err := sender.Delete().Messages(ctx, q.IDs...)
		return err
	}

	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return err
	}
	_, err = sender.To(resolved.InputPeer).Revoke().Messages(ctx, q.IDs...)
	return err
}

// PinMessage performs the Telegram RPC for `tg msg pin` and `tg msg unpin`.
func PinMessage(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.PinQuery) error {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return err
	}
	_, err = api.MessagesUpdatePinnedMessage(ctx, &tg.MessagesUpdatePinnedMessageRequest{
		Peer:   resolved.InputPeer,
		ID:     q.MessageID,
		Silent: q.Silent,
		Unpin:  q.Unpin,
	})
	return err
}

// ReactMessage performs the Telegram RPC for `tg msg react`.
func ReactMessage(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.ReactQuery) (output.SendResultRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.SendResultRow{}, err
	}
	var reactions []tg.ReactionClass
	if !q.Clear {
		reactions = []tg.ReactionClass{&tg.ReactionEmoji{Emoticon: q.Emoji}}
	}
	if _, err := gotdmessage.NewSender(api).To(resolved.InputPeer).Reaction(ctx, q.MessageID, reactions...); err != nil {
		return output.SendResultRow{}, err
	}
	return output.SendResultRow{Action: "react", MessageID: q.MessageID, ChatID: resolved.ID}, nil
}

// DownloadMessageMedia downloads the media attached to one message.
func DownloadMessageMedia(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.DownloadQuery) (output.DownloadRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.DownloadRow{}, err
	}
	elem, err := getMessageByID(ctx, api, resolved.InputPeer, q.Message)
	if err != nil {
		return output.DownloadRow{}, err
	}
	file, ok := elem.File()
	if !ok {
		return output.DownloadRow{}, telegrammessage.ErrNoMedia
	}
	path, err := mediaOutputPath(q.Output, file.Name)
	if err != nil {
		return output.DownloadRow{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return output.DownloadRow{}, err
	}
	if _, err := downloader.NewDownloader().Download(api, file.Location).ToPath(ctx, path); err != nil {
		return output.DownloadRow{}, err
	}
	var size int64
	if st, err := os.Stat(path); err == nil {
		size = st.Size()
	}
	return output.DownloadRow{
		MessageRef: ref.FormatMessageRef(output.PreferredRefFromResolved(resolved), q.Message),
		Path:       path,
		Name:       file.Name,
		MIMEType:   file.MIMEType,
		Bytes:      size,
	}, nil
}

func getMessageByID(ctx context.Context, api *tg.Client, input tg.InputPeerClass, id int) (msgquery.Elem, error) {
	iter := query.Messages(api).GetHistory(input).OffsetID(id + 1).BatchSize(1).Iter()
	if iter.Next(ctx) {
		elem := iter.Value()
		if elem.Msg.GetID() == id {
			return elem, nil
		}
	}
	if err := iter.Err(); err != nil {
		return msgquery.Elem{}, err
	}
	return msgquery.Elem{}, telegrammessage.ErrNotFound
}

func mediaOutputPath(outputPath, remoteName string) (string, error) {
	name := safeMediaName(remoteName)
	if outputPath == "" {
		return availableMediaPath(name), nil
	}
	if st, err := os.Stat(outputPath); err == nil && st.IsDir() {
		return availableMediaPath(filepath.Join(outputPath, name)), nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	return filepath.Clean(outputPath), nil
}

func safeMediaName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == string(os.PathSeparator) {
		return "media"
	}
	return name
}

func availableMediaPath(path string) string {
	path = filepath.Clean(path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", stem, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

// CancelScheduledMessages performs the Telegram RPC for `tg msg schedule-cancel`.
func CancelScheduledMessages(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.CancelScheduledQuery) error {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return err
	}
	_, err = gotdmessage.NewSender(api).To(resolved.InputPeer).Scheduled().Delete(ctx, q.IDs[0], q.IDs[1:]...)
	return err
}

// ListScheduledMessages performs the Telegram RPC for `tg msg schedule-list`.
func ListScheduledMessages(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.ScheduledListQuery) ([]output.ScheduledMessageRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}
	mm, err := gotdmessage.NewSender(api).To(resolved.InputPeer).Scheduled().History(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]output.ScheduledMessageRow, 0, len(mm.GetMessages()))
	for _, m := range mm.GetMessages() {
		msg, ok := m.(*tg.Message)
		if !ok {
			continue
		}
		rows = append(rows, output.ScheduledMessageRow{
			ID:           msg.ID,
			ScheduledFor: time.Unix(int64(msg.Date), 0).UTC().Format(time.RFC3339),
			Text:         msg.Message,
		})
	}
	return rows, nil
}

// ResolveMessageLinkPeer resolves the peer metadata needed for a t.me link.
func ResolveMessageLinkPeer(ctx context.Context, resolver *peer.Resolver, q actionmessage.LinkQuery) (actionmessage.LinkPeer, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return actionmessage.LinkPeer{}, err
	}
	if ch, ok := resolved.InputPeer.(*tg.InputPeerChannel); ok {
		return actionmessage.LinkPeer{
			Username:  resolved.Username,
			ChannelID: ch.ChannelID,
			IsChannel: true,
		}, nil
	}
	return actionmessage.LinkPeer{Username: resolved.Username}, nil
}

func captionStyling(caption, parse string, errOut io.Writer) []styling.StyledTextOption {
	if caption == "" {
		return nil
	}
	switch parse {
	case "html":
		return []styling.StyledTextOption{html.String(nil, caption)}
	case "markdown":
		_, _ = fmt.Fprintln(errOut, "--parse not supported by current gotd build; sending as plain text")
		return []styling.StyledTextOption{styling.Plain(caption)}
	default:
		return []styling.StyledTextOption{styling.Plain(caption)}
	}
}

func sentMessageRows(action string, upd tg.UpdatesClass) []output.SendResultRow {
	rows := sentMessages(upd)
	if len(rows) == 0 {
		return []output.SendResultRow{{Action: action}}
	}
	for i := range rows {
		rows[i].Action = action
	}
	return rows
}

func sentMessages(upd tg.UpdatesClass) []output.SendResultRow {
	if upd == nil {
		return nil
	}
	var rows []output.SendResultRow
	fromMsg := func(m *tg.Message) {
		rows = append(rows, output.SendResultRow{
			MessageID: m.ID,
			ChatID:    peerToID(m.PeerID),
			Date:      fmtUnix(m.Date),
		})
	}
	walkUpdates := func(updates []tg.UpdateClass) {
		for _, u := range updates {
			switch v := u.(type) {
			case *tg.UpdateNewMessage:
				if m, ok := v.Message.(*tg.Message); ok {
					fromMsg(m)
				}
			case *tg.UpdateNewChannelMessage:
				if m, ok := v.Message.(*tg.Message); ok {
					fromMsg(m)
				}
			case *tg.UpdateMessageID:
				rows = append(rows, output.SendResultRow{MessageID: v.ID})
			}
		}
	}

	switch v := upd.(type) {
	case *tg.Updates:
		walkUpdates(v.Updates)
	case *tg.UpdatesCombined:
		walkUpdates(v.Updates)
	case *tg.UpdateShortMessage:
		rows = append(rows, output.SendResultRow{MessageID: v.ID, ChatID: v.UserID, Date: fmtUnix(v.Date)})
	case *tg.UpdateShortSentMessage:
		rows = append(rows, output.SendResultRow{MessageID: v.ID, Date: fmtUnix(v.Date)})
	}
	if len(rows) == 0 {
		if id, err := unpack.MessageID(upd, nil); err == nil {
			rows = append(rows, output.SendResultRow{MessageID: id})
		}
	}
	return dedupeSentMessages(rows)
}

func dedupeSentMessages(rows []output.SendResultRow) []output.SendResultRow {
	if len(rows) < 2 {
		return rows
	}
	seen := make(map[int]int, len(rows))
	out := make([]output.SendResultRow, 0, len(rows))
	for _, row := range rows {
		if row.MessageID <= 0 {
			out = append(out, row)
			continue
		}
		if idx, ok := seen[row.MessageID]; ok {
			out[idx] = richerSendResult(out[idx], row)
			continue
		}
		seen[row.MessageID] = len(out)
		out = append(out, row)
	}
	return out
}

func richerSendResult(a, b output.SendResultRow) output.SendResultRow {
	if a.ChatID == 0 && b.ChatID != 0 {
		a.ChatID = b.ChatID
	}
	if a.Date == "" && b.Date != "" {
		a.Date = b.Date
	}
	return a
}

func peerToID(p tg.PeerClass) int64 {
	switch v := p.(type) {
	case *tg.PeerUser:
		return v.UserID
	case *tg.PeerChat:
		return -v.ChatID
	case *tg.PeerChannel:
		return -1_000_000_000_000 - v.ChannelID
	}
	return 0
}

func fmtUnix(ts int) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(int64(ts), 0).UTC().Format(time.RFC3339)
}
