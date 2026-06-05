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

// DeleteChat performs the RPC for `tg chat delete`. A supergroup/channel is
// deleted via channels.deleteChannel (irreversible, creator-only). A user/bot
// DM is removed from the account via messages.deleteHistory (with q.Revoke it
// also deletes the history on the other side). Basic groups are not handled —
// use `tg chat leave`.
func DeleteChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.DeleteChatQuery) (output.PeerRef, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.PeerRef{}, err
	}
	if inCh, ok := inputChannelFromPeer(resolved.InputPeer); ok {
		if _, err := api.ChannelsDeleteChannel(ctx, inCh); err != nil {
			return output.PeerRef{}, err
		}
		return output.PeerRefFromResolved(resolved), nil
	}
	if resolved.Kind == "user" || resolved.Kind == "bot" {
		if _, err := api.MessagesDeleteHistory(ctx, &tg.MessagesDeleteHistoryRequest{
			Peer:   resolved.InputPeer,
			Revoke: q.Revoke,
		}); err != nil {
			return output.PeerRef{}, err
		}
		return output.PeerRefFromResolved(resolved), nil
	}
	return output.PeerRef{}, fmt.Errorf("%w: only channels, supergroups, and user DMs can be deleted by ref (use `tg chat leave` for basic groups)", command.ErrUnsupported)
}
