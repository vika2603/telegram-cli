package telegram

import (
	"context"
	"strconv"
	"time"
	"unicode/utf16"

	msgpeer "github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/tg"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// ListMessages loads message history rows for a resolved peer query.
func ListMessages(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.ListQuery) ([]output.MessageRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}
	var rows []output.MessageRow
	baseRef := output.PreferredRefFromResolved(resolved)
	hist := query.Messages(api).GetHistory(resolved.InputPeer).BatchSize(100)
	if q.OffsetID > 0 {
		hist = hist.OffsetID(q.OffsetID)
	}
	iter := hist.Iter()
	for iter.Next(ctx) {
		if len(rows) >= q.Limit {
			break
		}
		el := iter.Value()
		m, ok := el.Msg.(*tg.Message)
		if !ok {
			continue
		}
		// gotd's GetHistory iterator walks newest -> oldest, so once we cross
		// MinID we can stop early instead of paging further.
		if q.MinID > 0 && m.ID <= q.MinID {
			break
		}
		t := time.Unix(int64(m.Date), 0)
		if !q.MinDate.IsZero() && t.Before(q.MinDate) {
			break
		}
		if !q.MaxDate.IsZero() && t.After(q.MaxDate) {
			continue
		}
		rows = append(rows, messageToRow(m, el.Entities, baseRef))
	}
	if q.Asc {
		reverseMessageRows(rows)
	}
	return rows, iter.Err()
}

func reverseMessageRows(rows []output.MessageRow) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}

func messageToRow(m *tg.Message, entities msgpeer.Entities, baseRef string) output.MessageRow {
	row := output.MessageRow{
		ID:       m.ID,
		Ref:      ref.FormatMessageRef(baseRef, m.ID),
		Date:     time.Unix(int64(m.Date), 0).UTC().Format(time.RFC3339),
		Text:     m.Message,
		IsPinned: m.Pinned,
	}
	if ed, ok := m.GetEditDate(); ok && ed > 0 {
		row.EditDate = time.Unix(int64(ed), 0).UTC().Format(time.RFC3339)
	}
	if gid, ok := m.GetGroupedID(); ok && gid != 0 {
		row.GroupedID = gid
	}
	if media, ok := m.GetMedia(); ok && media != nil {
		row.HasMedia = true
		row.MediaKind = searchMessageMediaKind(media)
	}
	if v, ok := m.GetViews(); ok {
		row.Views = v
	}
	if from, ok := m.GetFromID(); ok {
		fillMessageSender(&row, from, entities)
	} else if peer := m.GetPeerID(); peer != nil {
		fillMessageSender(&row, peer, entities)
	}
	if replyHeader, ok := m.GetReplyTo(); ok {
		if h, ok := replyHeader.(*tg.MessageReplyHeader); ok {
			if id, hasID := h.GetReplyToMsgID(); hasID {
				row.ReplyToID = id
			}
		}
	}
	if fwd, ok := m.GetFwdFrom(); ok {
		row.Forward = extractForward(fwd, entities)
	}
	if ents := m.Entities; len(ents) > 0 {
		row.Entities = extractEntities(m.Message, ents)
	}
	if rm, ok := m.GetReplyMarkup(); ok {
		row.Buttons = extractButtons(rm)
	}
	if reactions, ok := m.GetReactions(); ok {
		row.Reactions = extractReactions(reactions)
	}
	return row
}

// extractReactions flattens tg.MessageReactions into a stable
// output.ReactionCount slice. Empty input yields nil so omitempty
// drops the field; the alternative (empty slice) survives MarshalJSON
// as `"reactions":[]` and pollutes scripts. Reaction kinds the gotd
// schema knows about are bucketed via Kind for agents that want to
// branch without parsing strings.
func extractReactions(r tg.MessageReactions) []output.ReactionCount {
	if len(r.Results) == 0 {
		return nil
	}
	out := make([]output.ReactionCount, 0, len(r.Results))
	for _, rc := range r.Results {
		item := output.ReactionCount{Count: rc.Count}
		if _, hasOrder := rc.GetChosenOrder(); hasOrder {
			item.SelfReacted = true
		}
		switch v := rc.Reaction.(type) {
		case *tg.ReactionEmoji:
			item.Kind = "emoji"
			item.Emoji = v.Emoticon
		case *tg.ReactionCustomEmoji:
			item.Kind = "custom_emoji"
			item.CustomEmojiID = v.DocumentID
		case *tg.ReactionEmpty:
			item.Kind = "empty"
		default:
			// Newer reaction kinds (e.g. paid star reactions) may exist
			// in future gotd versions; surface them as "unknown" so the
			// reaction is still counted, then add a typed case here
			// once gotd is bumped.
			item.Kind = "unknown"
		}
		out = append(out, item)
	}
	return out
}

func fillMessageSender(row *output.MessageRow, p tg.PeerClass, entities msgpeer.Entities) {
	row.FromID = peerID(p)
	switch v := p.(type) {
	case *tg.PeerUser:
		row.FromKind = "user"
		if u, ok := entities.User(v.UserID); ok {
			row.FromTitle = userTitle(u)
			row.FromUsername = u.Username
			if u.Bot {
				row.FromKind = "bot"
			}
			row.FromRef = output.PreferredPeerRef(row.FromKind, row.FromUsername, u.ID, u.AccessHash, false)
		}
	case *tg.PeerChat:
		row.FromKind = "chat"
		row.FromRef = output.PreferredPeerRef(row.FromKind, "", v.ChatID, 0, false)
		if ch, ok := entities.Chat(v.ChatID); ok {
			row.FromTitle = ch.Title
		}
	case *tg.PeerChannel:
		row.FromKind = "channel"
		if ch, ok := entities.Channel(v.ChannelID); ok {
			row.FromTitle = ch.Title
			row.FromUsername = ch.Username
			if !ch.Broadcast {
				row.FromKind = "chat"
			}
			row.FromRef = output.PreferredPeerRef(row.FromKind, row.FromUsername, ch.ID, ch.AccessHash, true)
		}
	}
}

func peerID(p tg.PeerClass) int64 {
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

// extractForward maps a MessageFwdHeader into the output ForwardInfo, resolving
// the origin peer through the response Entities collection when available.
func extractForward(fwd tg.MessageFwdHeader, entities msgpeer.Entities) *output.ForwardInfo {
	info := &output.ForwardInfo{}
	if fwd.Date > 0 {
		info.Date = time.Unix(int64(fwd.Date), 0).UTC().Format(time.RFC3339)
	}
	if name, ok := fwd.GetFromName(); ok {
		info.FromName = name
	}
	if post, ok := fwd.GetChannelPost(); ok {
		info.ChannelPostID = post
	}
	if author, ok := fwd.GetPostAuthor(); ok {
		info.PostAuthor = author
	}
	if origin, ok := fwd.GetFromID(); ok {
		info.From = forwardPeerObject(origin, entities)
		if info.From != nil && info.From.Username != "" && info.ChannelPostID > 0 {
			info.Link = "https://t.me/" + info.From.Username + "/" + strconv.Itoa(info.ChannelPostID)
		}
	}
	if info.From == nil && info.FromName == "" && info.ChannelPostID == 0 && info.Date == "" {
		return nil
	}
	return info
}

func forwardPeerObject(p tg.PeerClass, entities msgpeer.Entities) *output.PeerObject {
	obj := &output.PeerObject{ID: peerID(p)}
	switch v := p.(type) {
	case *tg.PeerUser:
		obj.Type = "user"
		if u, ok := entities.User(v.UserID); ok {
			obj.Title = userTitle(u)
			obj.Username = u.Username
			if u.Bot {
				obj.Type = "bot"
			}
			obj.Ref = output.PreferredPeerRef(obj.Type, obj.Username, u.ID, u.AccessHash, false)
		}
	case *tg.PeerChat:
		obj.Type = "chat"
		obj.Ref = output.PreferredPeerRef(obj.Type, "", v.ChatID, 0, false)
		if c, ok := entities.Chat(v.ChatID); ok {
			obj.Title = c.Title
		}
	case *tg.PeerChannel:
		obj.Type = "channel"
		if c, ok := entities.Channel(v.ChannelID); ok {
			obj.Title = c.Title
			obj.Username = c.Username
			if !c.Broadcast {
				obj.Type = "chat"
			}
			obj.Ref = output.PreferredPeerRef(obj.Type, obj.Username, c.ID, c.AccessHash, true)
		}
	default:
		return nil
	}
	if obj.Username != "" {
		obj.Link = "https://t.me/" + obj.Username
	}
	return obj
}

// extractButtons flattens inline-keyboard markup into MessageButton rows.
// Non-inline keyboards (reply keyboards) are ignored.
func extractButtons(rm tg.ReplyMarkupClass) []output.MessageButton {
	inline, ok := rm.(*tg.ReplyInlineMarkup)
	if !ok {
		return nil
	}
	var out []output.MessageButton
	for i, row := range inline.Rows {
		for _, b := range row.Buttons {
			switch v := b.(type) {
			case *tg.KeyboardButtonURL:
				out = append(out, output.MessageButton{Row: i, Text: v.Text, URL: v.URL, Type: "url"})
			case *tg.KeyboardButtonURLAuth:
				out = append(out, output.MessageButton{Row: i, Text: v.Text, URL: v.URL, Type: "url_auth"})
			case *tg.KeyboardButtonSwitchInline:
				out = append(out, output.MessageButton{Row: i, Text: v.Text, Type: "switch_inline"})
			case *tg.KeyboardButtonCallback:
				out = append(out, output.MessageButton{Row: i, Text: v.Text, Type: "callback"})
			case *tg.KeyboardButtonWebView:
				out = append(out, output.MessageButton{Row: i, Text: v.Text, URL: v.URL, Type: "web_app"})
			case *tg.KeyboardButton:
				out = append(out, output.MessageButton{Row: i, Text: v.Text, Type: "text"})
			}
		}
	}
	return out
}

// extractEntities flattens Telegram message entities into the JSON-friendly
// shape, slicing the message text by UTF-16 offsets as required by the schema
// (https://core.telegram.org/api/entities#entity-length).
func extractEntities(text string, ents []tg.MessageEntityClass) []output.MessageEntity {
	if len(ents) == 0 {
		return nil
	}
	units := utf16.Encode([]rune(text))
	slice := func(offset, length int) string {
		if offset < 0 || length <= 0 || offset+length > len(units) {
			return ""
		}
		return string(utf16.Decode(units[offset : offset+length]))
	}
	out := make([]output.MessageEntity, 0, len(ents))
	for _, e := range ents {
		switch v := e.(type) {
		case *tg.MessageEntityTextURL:
			out = append(out, output.MessageEntity{Type: "text_url", Text: slice(v.Offset, v.Length), URL: v.URL})
		case *tg.MessageEntityURL:
			out = append(out, output.MessageEntity{Type: "url", Text: slice(v.Offset, v.Length), URL: slice(v.Offset, v.Length)})
		case *tg.MessageEntityMention:
			out = append(out, output.MessageEntity{Type: "mention", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityMentionName:
			out = append(out, output.MessageEntity{Type: "mention_name", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityHashtag:
			out = append(out, output.MessageEntity{Type: "hashtag", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityCashtag:
			out = append(out, output.MessageEntity{Type: "cashtag", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityBotCommand:
			out = append(out, output.MessageEntity{Type: "bot_command", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityEmail:
			out = append(out, output.MessageEntity{Type: "email", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityPhone:
			out = append(out, output.MessageEntity{Type: "phone", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityCode:
			out = append(out, output.MessageEntity{Type: "code", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityPre:
			out = append(out, output.MessageEntity{Type: "pre", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityBold:
			out = append(out, output.MessageEntity{Type: "bold", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityItalic:
			out = append(out, output.MessageEntity{Type: "italic", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityUnderline:
			out = append(out, output.MessageEntity{Type: "underline", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityStrike:
			out = append(out, output.MessageEntity{Type: "strike", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntitySpoiler:
			out = append(out, output.MessageEntity{Type: "spoiler", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityBlockquote:
			out = append(out, output.MessageEntity{Type: "blockquote", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityBankCard:
			out = append(out, output.MessageEntity{Type: "bank_card", Text: slice(v.Offset, v.Length)})
		case *tg.MessageEntityCustomEmoji:
			out = append(out, output.MessageEntity{Type: "custom_emoji", Text: slice(v.Offset, v.Length)})
		}
	}
	return out
}
