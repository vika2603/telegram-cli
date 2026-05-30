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

// DeleteChat performs the RPC for `tg chat delete`: it deletes a supergroup or
// channel via channels.deleteChannel (irreversible, creator-only).
func DeleteChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.DeleteChatQuery) (output.PeerRef, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.PeerRef{}, err
	}
	inCh, ok := inputChannelFromPeer(resolved.InputPeer)
	if !ok {
		return output.PeerRef{}, fmt.Errorf("%w: only supergroups and channels can be deleted by ref", command.ErrUsage)
	}
	if _, err := api.ChannelsDeleteChannel(ctx, inCh); err != nil {
		return output.PeerRef{}, err
	}
	return output.PeerRefFromResolved(resolved), nil
}
