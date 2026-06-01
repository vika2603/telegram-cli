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

// SetChannelDiscussion links or unlinks a broadcast channel's discussion
// supergroup via channels.setDiscussionGroup. Unlink passes InputChannelEmpty.
func SetChannelDiscussion(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.DiscussionQuery) (output.DiscussionRow, error) {
	chResolved, err := resolver.Resolve(ctx, q.Channel)
	if err != nil {
		return output.DiscussionRow{}, err
	}
	broadcast, ok := inputChannelFromPeer(chResolved.InputPeer)
	if !ok {
		return output.DiscussionRow{}, fmt.Errorf("%w: a discussion group can only be linked to a channel", command.ErrUnsupported)
	}

	var group tg.InputChannelClass = &tg.InputChannelEmpty{}
	row := output.DiscussionRow{Action: "unlink", Channel: output.PeerRefFromResolved(chResolved)}
	if !q.Unlink {
		grpResolved, err := resolver.Resolve(ctx, q.Group)
		if err != nil {
			return output.DiscussionRow{}, err
		}
		grpCh, ok := inputChannelFromPeer(grpResolved.InputPeer)
		if !ok {
			return output.DiscussionRow{}, fmt.Errorf("%w: the discussion group must be a supergroup", command.ErrUnsupported)
		}
		group = grpCh
		row.Action = "link"
		gref := output.PeerRefFromResolved(grpResolved)
		row.Group = &gref
	}

	if _, err := api.ChannelsSetDiscussionGroup(ctx, &tg.ChannelsSetDiscussionGroupRequest{
		Broadcast: broadcast,
		Group:     group,
	}); err != nil {
		return output.DiscussionRow{}, err
	}
	return row, nil
}

// ListDiscussionCandidates returns supergroups eligible to be linked as a
// channel's discussion group (channels.getGroupsForDiscussion).
func ListDiscussionCandidates(ctx context.Context, api *tg.Client) ([]output.ChatRow, error) {
	res, err := api.ChannelsGetGroupsForDiscussion(ctx)
	if err != nil {
		return nil, err
	}
	var chats []tg.ChatClass
	switch v := res.(type) {
	case *tg.MessagesChats:
		chats = v.Chats
	case *tg.MessagesChatsSlice:
		chats = v.Chats
	}
	rows := make([]output.ChatRow, 0, len(chats))
	for _, c := range chats {
		pr := output.PeerRefFromChat(c)
		if pr.ID == 0 {
			continue
		}
		rows = append(rows, output.ChatRow{
			ID:       pr.ID,
			Ref:      pr.Ref,
			Kind:     pr.Kind,
			Title:    pr.Title,
			Username: pr.Username,
		})
	}
	return rows, nil
}
