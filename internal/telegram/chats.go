package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	actionsearch "github.com/vika2603/telegram-cli/internal/action/search"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// ShowChat resolves a peer reference and returns the row rendered by `tg chat show`.
func ShowChat(ctx context.Context, resolver *peer.Resolver, q actionchat.ShowQuery) (output.ChatRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.ChatRow{}, err
	}
	return output.ChatRow{
		ID:       resolved.ID,
		Ref:      output.PreferredRefFromResolved(resolved),
		Kind:     resolved.Kind,
		Title:    resolved.Title,
		Username: resolved.Username,
	}, nil
}

// SearchChats loads chat search rows through contacts.search.
func SearchChats(ctx context.Context, api *tg.Client, q actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
	resp, err := api.ContactsSearch(ctx, &tg.ContactsSearchRequest{
		Q:     q.Query,
		Limit: q.Limit,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]output.SearchChatRow, 0, len(resp.MyResults)+len(resp.Results))
	seen := map[string]bool{}
	add := func(p tg.PeerClass, source string) {
		row := searchPeerToChatRow(p, resp)
		if row.ID == 0 {
			return
		}
		key := fmt.Sprintf("%s:%d", row.Kind, row.ID)
		if seen[key] {
			return
		}
		seen[key] = true
		rows = append(rows, output.SearchChatRow{ChatRow: row, Source: source})
	}
	for _, p := range resp.MyResults {
		add(p, "my")
	}
	for _, p := range resp.Results {
		add(p, "public")
	}
	return rows, nil
}

// searchPeerToChatRow resolves a PeerClass against the entities returned by
// contacts.search. Linear scans over resp.Users/resp.Chats are fine here
// because ContactsSearch is server-capped at a small result set.
func searchPeerToChatRow(p tg.PeerClass, resp *tg.ContactsFound) output.ChatRow {
	row := output.ChatRow{}
	switch v := p.(type) {
	case *tg.PeerUser:
		row.ID = v.UserID
		row.Kind = "user"
		for _, u := range resp.Users {
			if user, ok := u.(*tg.User); ok && user.ID == v.UserID {
				row.Title = userTitle(user)
				row.Username = user.Username
				if user.Bot {
					row.Kind = "bot"
				}
				row.Ref = output.PreferredPeerRef(row.Kind, row.Username, user.ID, user.AccessHash, false)
				break
			}
		}
		if row.Ref == "" {
			row.Ref = output.PreferredPeerRef(row.Kind, "", v.UserID, 0, false)
		}
	case *tg.PeerChat:
		row.ID = -v.ChatID
		row.Kind = "chat"
		row.Ref = output.PreferredPeerRef(row.Kind, "", v.ChatID, 0, false)
		for _, c := range resp.Chats {
			if ch, ok := c.(*tg.Chat); ok && ch.ID == v.ChatID {
				row.Title = ch.Title
				break
			}
		}
	case *tg.PeerChannel:
		row.ID = -1_000_000_000_000 - v.ChannelID
		row.Kind = "channel"
		for _, c := range resp.Chats {
			if ch, ok := c.(*tg.Channel); ok && ch.ID == v.ChannelID {
				row.Title = ch.Title
				row.Username = ch.Username
				if !ch.Broadcast {
					row.Kind = "chat"
				}
				row.Ref = output.PreferredPeerRef(row.Kind, row.Username, ch.ID, ch.AccessHash, true)
				break
			}
		}
		if row.Ref == "" {
			row.Ref = output.PreferredPeerRef(row.Kind, "", v.ChannelID, 0, true)
		}
	}
	return row
}
