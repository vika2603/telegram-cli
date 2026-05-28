package telegram

import (
	"context"
	"sort"
	"time"

	msgpeer "github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/telegram/query"
	msgquery "github.com/gotd/td/telegram/query/messages"
	"github.com/gotd/td/tg"

	actionsearch "github.com/vika2603/telegram-cli/internal/action/search"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// SearchMessages loads message search rows through gotd.
func SearchMessages(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionsearch.MessageQuery) ([]output.SearchMsgRow, error) {
	filter := searchMessageFilter(q.Filter, q.Missed)
	localPoll := q.Filter == actionsearch.MessageFilterPoll
	var rows []output.SearchMsgRow

	appendElem := func(el msgquery.Elem, fallback peer.Resolved) {
		row, ok := searchMessageElemToRow(el, fallback)
		if !ok {
			return
		}
		if localPoll && row.MediaKind != "poll" {
			return
		}
		if q.GroupsOnly && row.ChatKind != "chat" {
			return
		}
		if q.UsersOnly && row.ChatKind != "user" && row.ChatKind != "bot" {
			return
		}
		if q.BroadcastsOnly && row.ChatKind != "channel" {
			return
		}
		rows = append(rows, row)
	}

	if q.InRef != nil {
		resolved, err := resolver.Resolve(ctx, *q.InRef)
		if err != nil {
			return nil, err
		}
		b := query.Messages(api).Search(resolved.InputPeer).
			Q(q.Query).
			Filter(filter).
			BatchSize(100)
		if !q.MinDate.IsZero() {
			b = b.MinDate(int(q.MinDate.Unix()))
		}
		if !q.MaxDate.IsZero() {
			b = b.MaxDate(int(q.MaxDate.Unix()))
		}
		if q.FromRef != nil {
			from, err := resolver.Resolve(ctx, *q.FromRef)
			if err != nil {
				return nil, err
			}
			b = b.FromID(from.InputPeer)
		}
		iter := b.Iter()
		for iter.Next(ctx) {
			if len(rows) >= q.Limit {
				break
			}
			appendElem(iter.Value(), resolved)
			if len(rows) >= q.Limit {
				break
			}
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
	} else {
		b := query.Messages(api).SearchGlobal().
			Q(q.Query).
			Filter(filter).
			BatchSize(100)
		if q.BroadcastsOnly {
			b = b.BroadcastsOnly(true)
		}
		if !q.MinDate.IsZero() {
			b = b.MinDate(int(q.MinDate.Unix()))
		}
		if !q.MaxDate.IsZero() {
			b = b.MaxDate(int(q.MaxDate.Unix()))
		}
		iter := b.Iter()
		for iter.Next(ctx) {
			if len(rows) >= q.Limit {
				break
			}
			appendElem(iter.Value(), peer.Resolved{})
			if len(rows) >= q.Limit {
				break
			}
		}
		if err := iter.Err(); err != nil {
			return nil, err
		}
	}

	if q.Asc {
		sort.SliceStable(rows, func(i, j int) bool {
			return rows[i].Date < rows[j].Date
		})
	}
	return rows, nil
}

func searchMessageFilter(name actionsearch.MessageFilter, missed bool) tg.MessagesFilterClass {
	switch name {
	case actionsearch.MessageFilterPhotos:
		return &tg.InputMessagesFilterPhotos{}
	case actionsearch.MessageFilterVideo:
		return &tg.InputMessagesFilterVideo{}
	case actionsearch.MessageFilterDocument:
		return &tg.InputMessagesFilterDocument{}
	case actionsearch.MessageFilterVoice:
		return &tg.InputMessagesFilterVoice{}
	case actionsearch.MessageFilterMusic:
		return &tg.InputMessagesFilterMusic{}
	case actionsearch.MessageFilterGIF:
		return &tg.InputMessagesFilterGif{}
	case actionsearch.MessageFilterURL:
		return &tg.InputMessagesFilterURL{}
	case actionsearch.MessageFilterPinned:
		return &tg.InputMessagesFilterPinned{}
	case actionsearch.MessageFilterGeo:
		return &tg.InputMessagesFilterGeo{}
	case actionsearch.MessageFilterMyMentions:
		return &tg.InputMessagesFilterMyMentions{}
	case actionsearch.MessageFilterRoundVideo:
		return &tg.InputMessagesFilterRoundVideo{}
	case actionsearch.MessageFilterRoundVoice:
		return &tg.InputMessagesFilterRoundVoice{}
	case actionsearch.MessageFilterPhoneCalls:
		return &tg.InputMessagesFilterPhoneCalls{Missed: missed}
	case actionsearch.MessageFilterChatPhotos:
		return &tg.InputMessagesFilterChatPhotos{}
	case actionsearch.MessageFilterContacts:
		return &tg.InputMessagesFilterContacts{}
	case actionsearch.MessageFilterPhotoVideo:
		return &tg.InputMessagesFilterPhotoVideo{}
	default:
		return &tg.InputMessagesFilterEmpty{}
	}
}

func searchMessageElemToRow(el msgquery.Elem, fallback peer.Resolved) (output.SearchMsgRow, bool) {
	row := output.SearchMsgRow{
		MessageID: el.Msg.GetID(),
		Date:      time.Unix(int64(el.Msg.GetDate()), 0).UTC().Format(time.RFC3339),
	}

	if m, ok := el.Msg.(*tg.Message); ok {
		// Reuse messageToRow so PR #2's forward / entities / buttons and
		// the rest of the message metadata (reply target, views, pinned,
		// resolved sender ref / kind / title / username) populate the
		// same way `tg msg list` populates them. Previously this path
		// rolled its own row and dropped every one of those fields.
		inner := messageToRow(m, el.Entities, "")
		row.EditDate = inner.EditDate
		row.GroupedID = inner.GroupedID
		row.FromID = inner.FromID
		row.FromRef = inner.FromRef
		row.FromKind = inner.FromKind
		row.FromTitle = inner.FromTitle
		row.FromUsername = inner.FromUsername
		row.Text = inner.Text
		row.ReplyToID = inner.ReplyToID
		row.Entities = inner.Entities
		row.Buttons = inner.Buttons
		row.Forward = inner.Forward
		row.Reactions = inner.Reactions
		row.HasMedia = inner.HasMedia
		row.MediaKind = inner.MediaKind
		row.Views = inner.Views
		row.IsPinned = inner.IsPinned
	} else if from, ok := el.Msg.GetFromID(); ok {
		// MessageService and friends carry only the sender id.
		row.FromID = peerID(from)
	}

	if fallback.ID != 0 {
		row.ChatID = fallback.ID
		row.ChatTitle = fallback.Title
		row.ChatKind = fallback.Kind
	} else {
		row.ChatID, row.ChatKind, row.ChatTitle = searchMessagePeerDetails(el.Msg.GetPeerID(), el.Entities)
	}
	if row.ChatID == 0 {
		row.ChatID = peerID(el.Msg.GetPeerID())
	}
	return row, row.MessageID != 0
}

func searchMessagePeerDetails(p tg.PeerClass, entities msgpeer.Entities) (id int64, kind, title string) {
	switch v := p.(type) {
	case *tg.PeerUser:
		id = v.UserID
		kind = "user"
		if u, ok := entities.User(v.UserID); ok {
			title = userTitle(u)
			if u.Bot {
				kind = "bot"
			}
		}
	case *tg.PeerChat:
		id = -v.ChatID
		kind = "chat"
		if ch, ok := entities.Chat(v.ChatID); ok {
			title = ch.Title
		}
	case *tg.PeerChannel:
		id = -1_000_000_000_000 - v.ChannelID
		kind = "channel"
		if ch, ok := entities.Channel(v.ChannelID); ok {
			title = ch.Title
			if !ch.Broadcast {
				kind = "chat"
			}
		}
	}
	return id, kind, title
}

func searchMessageMediaKind(media tg.MessageMediaClass) string {
	switch v := media.(type) {
	case *tg.MessageMediaPhoto:
		return "photo"
	case *tg.MessageMediaDocument:
		if doc, ok := v.Document.AsNotEmpty(); ok {
			return searchMessageDocumentMediaKind(doc)
		}
		return "document"
	case *tg.MessageMediaWebPage:
		return "web_page"
	case *tg.MessageMediaGeo, *tg.MessageMediaGeoLive, *tg.MessageMediaVenue:
		return "geo"
	case *tg.MessageMediaContact:
		return "contact"
	case *tg.MessageMediaPoll:
		return "poll"
	default:
		return "other"
	}
}

func searchMessageDocumentMediaKind(doc *tg.Document) string {
	for _, attr := range doc.Attributes {
		switch v := attr.(type) {
		case *tg.DocumentAttributeSticker:
			return "sticker"
		case *tg.DocumentAttributeAnimated:
			return "gif"
		case *tg.DocumentAttributeVideo:
			if v.RoundMessage {
				return "round_video"
			}
			return "video"
		case *tg.DocumentAttributeAudio:
			if v.Voice {
				return "voice"
			}
			return "audio"
		}
	}
	return "document"
}
