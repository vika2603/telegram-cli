package telegram

import (
	"context"
	"time"

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
	return row
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
