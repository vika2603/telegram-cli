package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// EditChat performs the RPCs for `tg chat edit` / `tg channel edit`: it sets
// the title (channels.editTitle) and/or the about text (messages.editChatAbout)
// of a supergroup or channel, changing only the fields the caller passed.
func EditChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.EditChatQuery) (output.ChatRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.ChatRow{}, err
	}

	title := resolved.Title
	if q.Title != nil {
		inCh, ok := inputChannelFromPeer(resolved.InputPeer)
		if !ok {
			return output.ChatRow{}, fmt.Errorf("%w: only supergroups and channels can be edited by ref", command.ErrUsage)
		}
		if _, err := api.ChannelsEditTitle(ctx, &tg.ChannelsEditTitleRequest{Channel: inCh, Title: *q.Title}); err != nil {
			return output.ChatRow{}, err
		}
		title = *q.Title
	}
	if q.About != nil {
		if _, err := api.MessagesEditChatAbout(ctx, &tg.MessagesEditChatAboutRequest{Peer: resolved.InputPeer, About: *q.About}); err != nil {
			return output.ChatRow{}, err
		}
	}

	username := resolved.Username
	if q.Username != nil {
		inCh, ok := inputChannelFromPeer(resolved.InputPeer)
		if !ok {
			return output.ChatRow{}, fmt.Errorf("%w: only supergroups and channels have public usernames", command.ErrUsage)
		}
		// "" removes the public username (makes the chat private); a value
		// sets/replaces it (makes it public).
		if _, err := api.ChannelsUpdateUsername(ctx, &tg.ChannelsUpdateUsernameRequest{Channel: inCh, Username: *q.Username}); err != nil {
			return output.ChatRow{}, err
		}
		username = *q.Username
	}

	return output.ChatRow{
		ID:       resolved.ID,
		Ref:      output.PreferredRefFromResolved(resolved),
		Kind:     resolved.Kind,
		Title:    title,
		Username: username,
	}, nil
}
