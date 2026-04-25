package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/query"
	"github.com/gotd/td/telegram/query/dialogs"
	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/output"
)

// ListChats loads dialog rows through gotd.
func ListChats(ctx context.Context, api *tg.Client, req actionchat.ListRequest) ([]output.ChatRow, error) {
	var out []output.ChatRow
	iter := query.GetDialogs(api).BatchSize(100).Iter()
	for iter.Next(ctx) {
		if len(out) >= req.Limit {
			break
		}
		row, ok := dialogToRow(iter.Value())
		if !ok {
			continue
		}
		if req.ArchivedOnly && !row.IsArchived {
			continue
		}
		if req.PinnedOnly && !row.IsPinned {
			continue
		}
		out = append(out, row)
	}
	return out, iter.Err()
}

// dialogToRow converts a gotd dialogs.Elem to a ChatRow. Returns false for
// DialogFolder entries, which are not real dialogs.
func dialogToRow(el dialogs.Elem) (output.ChatRow, bool) {
	d, ok := el.Dialog.(*tg.Dialog)
	if !ok {
		return output.ChatRow{}, false
	}
	row := output.ChatRow{
		TopMessage:  d.TopMessage,
		UnreadCount: d.UnreadCount,
		IsPinned:    d.Pinned,
	}
	switch p := el.Peer.(type) {
	case *tg.InputPeerUser:
		row.ID = p.UserID
		row.Kind = "user"
		row.Ref = output.PreferredPeerRef(row.Kind, "", p.UserID, p.AccessHash, false)
		if u, ok := el.Entities.User(p.UserID); ok {
			row.Title = userTitle(u)
			row.Username = u.Username
			if u.Bot {
				row.Kind = "bot"
			}
			row.Ref = output.PreferredPeerRef(row.Kind, row.Username, p.UserID, p.AccessHash, false)
		}
	case *tg.InputPeerChat:
		row.ID = -p.ChatID
		row.Kind = "chat"
		row.Ref = output.PreferredPeerRef(row.Kind, "", p.ChatID, 0, false)
		if c, ok := el.Entities.Chat(p.ChatID); ok {
			row.Title = c.Title
		}
	case *tg.InputPeerChannel:
		row.ID = -1_000_000_000_000 - p.ChannelID
		row.Kind = "channel"
		row.Ref = output.PreferredPeerRef(row.Kind, "", p.ChannelID, p.AccessHash, true)
		if c, ok := el.Entities.Channel(p.ChannelID); ok {
			row.Title = c.Title
			row.Username = c.Username
			if !c.Broadcast {
				row.Kind = "chat"
			}
			row.Ref = output.PreferredPeerRef(row.Kind, row.Username, p.ChannelID, p.AccessHash, true)
		}
	}
	if m, ok := el.Last.(*tg.Message); ok && row.Ref != "" {
		summary := output.MessageSummaryFromRow(messageToRow(m, el.Entities, row.Ref))
		row.Last = &summary
	}
	return row, true
}

func userTitle(u *tg.User) string {
	switch {
	case u.FirstName != "" && u.LastName != "":
		return u.FirstName + " " + u.LastName
	case u.FirstName != "":
		return u.FirstName
	case u.LastName != "":
		return u.LastName
	case u.Username != "":
		return "@" + u.Username
	}
	return fmt.Sprintf("user#%d", u.ID)
}
