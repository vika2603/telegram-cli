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

	// Title, username, and the toggle settings all require an InputChannel
	// (supergroups/channels only). About and no-forwards work on the InputPeer
	// directly, so they don't force a channel. Resolve it once and reuse below.
	needsChannel := q.Title != nil || q.Username != nil || q.Forum != nil ||
		q.HideMembers != nil || q.HideHistory != nil || q.SlowMode != nil || q.Signatures != nil
	var inCh tg.InputChannelClass
	if needsChannel {
		var ok bool
		inCh, ok = inputChannelFromPeer(resolved.InputPeer)
		if !ok {
			return output.ChatRow{}, fmt.Errorf("%w: only supergroups and channels support this setting", command.ErrUnsupported)
		}
	}

	title := resolved.Title
	if q.Title != nil {
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
		// inCh is guaranteed non-nil here (checked above in needsChannel block).
		// "" removes the public username (makes the chat private); a value
		// sets/replaces it (makes it public).
		if _, err := api.ChannelsUpdateUsername(ctx, &tg.ChannelsUpdateUsernameRequest{Channel: inCh, Username: *q.Username}); err != nil {
			return output.ChatRow{}, err
		}
		username = *q.Username
	}

	// Toggle RPCs — all require an InputChannel (supergroups/channels only).
	if q.Forum != nil {
		if _, err := api.ChannelsToggleForum(ctx, &tg.ChannelsToggleForumRequest{Channel: inCh, Enabled: *q.Forum}); err != nil {
			return output.ChatRow{}, err
		}
	}
	if q.HideMembers != nil {
		if _, err := api.ChannelsToggleParticipantsHidden(ctx, &tg.ChannelsToggleParticipantsHiddenRequest{Channel: inCh, Enabled: *q.HideMembers}); err != nil {
			return output.ChatRow{}, err
		}
	}
	if q.HideHistory != nil {
		if _, err := api.ChannelsTogglePreHistoryHidden(ctx, &tg.ChannelsTogglePreHistoryHiddenRequest{Channel: inCh, Enabled: *q.HideHistory}); err != nil {
			return output.ChatRow{}, err
		}
	}
	if q.SlowMode != nil {
		if _, err := api.ChannelsToggleSlowMode(ctx, &tg.ChannelsToggleSlowModeRequest{Channel: inCh, Seconds: *q.SlowMode}); err != nil {
			return output.ChatRow{}, err
		}
	}
	if q.Signatures != nil {
		if _, err := api.ChannelsToggleSignatures(ctx, &tg.ChannelsToggleSignaturesRequest{Channel: inCh, Enabled: *q.Signatures}); err != nil {
			return output.ChatRow{}, err
		}
	}
	// NoForwards uses InputPeer directly, not InputChannel.
	if q.NoForwards != nil {
		if _, err := api.MessagesToggleNoForwards(ctx, &tg.MessagesToggleNoForwardsRequest{Peer: resolved.InputPeer, Enabled: *q.NoForwards}); err != nil {
			return output.ChatRow{}, err
		}
	}

	return output.ChatRow{
		ID:       resolved.ID,
		Ref:      output.PreferredRefFromResolved(resolved),
		Kind:     resolved.Kind,
		Title:    title,
		Username: username,
	}, nil
}
