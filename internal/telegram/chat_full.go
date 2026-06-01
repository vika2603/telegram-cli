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

// ShowChatFull resolves a chat and enriches it with getFullChannel data
// (about, member/admin/online counts, linked discussion group, slow mode,
// pinned message). Full info is only available for supergroups and channels.
func ShowChatFull(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.ShowQuery) (output.ChatRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.ChatRow{}, err
	}
	row := output.ChatRow{
		ID:       resolved.ID,
		Ref:      output.PreferredRefFromResolved(resolved),
		Kind:     resolved.Kind,
		Title:    resolved.Title,
		Username: resolved.Username,
	}

	inCh, ok := inputChannelFromPeer(resolved.InputPeer)
	if !ok {
		return output.ChatRow{}, fmt.Errorf("%w: --full is only available for supergroups and channels", command.ErrUnsupported)
	}
	full, err := api.ChannelsGetFullChannel(ctx, inCh)
	if err != nil {
		return output.ChatRow{}, err
	}
	cf, ok := full.FullChat.(*tg.ChannelFull)
	if !ok {
		return row, nil
	}
	row.About = cf.About
	if v, ok := cf.GetParticipantsCount(); ok {
		row.MembersCount = v
	}
	if v, ok := cf.GetAdminsCount(); ok {
		row.AdminsCount = v
	}
	if v, ok := cf.GetOnlineCount(); ok {
		row.OnlineCount = v
	}
	if v, ok := cf.GetPinnedMsgID(); ok {
		row.PinnedMsgID = v
	}
	if v, ok := cf.GetLinkedChatID(); ok {
		row.LinkedChatID = v
	}
	if v, ok := cf.GetSlowmodeSeconds(); ok {
		row.SlowmodeSeconds = v
	}
	return row, nil
}
