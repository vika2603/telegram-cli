package telegram

import (
	"context"
	"errors"

	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/output"
)

// CreateChat performs the RPC for `tg chat create`. It creates a supergroup
// (default), a broadcast channel (--channel), or a forum supergroup (--forum)
// via channels.createChannel, and returns the new peer as a ChatRow.
func CreateChat(ctx context.Context, api *tg.Client, q actionchat.CreateChatQuery) (output.ChatRow, error) {
	req := &tg.ChannelsCreateChannelRequest{Title: q.Title, About: q.About}
	if q.Broadcast {
		req.Broadcast = true
	} else {
		req.Megagroup = true
	}
	if q.Forum {
		req.Forum = true
	}
	upd, err := api.ChannelsCreateChannel(ctx, req)
	if err != nil {
		return output.ChatRow{}, err
	}
	ch, ok := firstChannelFromUpdates(upd)
	if !ok {
		return output.ChatRow{}, errors.New("create chat: server response carried no channel")
	}
	kind := "channel"
	if !ch.Broadcast {
		kind = "chat"
	}
	return output.ChatRow{
		ID:       -1_000_000_000_000 - ch.ID,
		Ref:      output.PreferredPeerRef(kind, ch.Username, ch.ID, ch.AccessHash, true),
		Kind:     kind,
		Title:    ch.Title,
		Username: ch.Username,
	}, nil
}

// firstChannelFromUpdates pulls the freshly created channel out of the
// Updates createChannel returns (it lands in the Chats list).
func firstChannelFromUpdates(upd tg.UpdatesClass) (*tg.Channel, bool) {
	var chats []tg.ChatClass
	switch v := upd.(type) {
	case *tg.Updates:
		chats = v.Chats
	case *tg.UpdatesCombined:
		chats = v.Chats
	}
	for _, c := range chats {
		if ch, ok := c.(*tg.Channel); ok {
			return ch, true
		}
	}
	return nil, false
}
