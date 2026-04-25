package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	chaterr "github.com/vika2603/telegram-cli/internal/telegram/chat"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// MoveChatToFolder resolves a chat and moves it to the requested folder.
func MoveChatToFolder(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.FolderQuery) (output.ChatFolderRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.ChatFolderRow{}, err
	}
	reqPeers := []tg.InputFolderPeer{{Peer: resolved.InputPeer, FolderID: q.Folder}}
	if _, err := api.FoldersEditPeerFolders(ctx, reqPeers); err != nil {
		return output.ChatFolderRow{}, err
	}
	return output.ChatFolderRow{
		Action: q.Action,
		Peer:   output.PeerRefFromResolved(resolved),
		Folder: q.Folder,
	}, nil
}

// MuteChat updates a chat notification mute-until value.
func MuteChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.MuteQuery) (output.ChatMuteRow, error) {
	row, err := updateChatMute(ctx, api, resolver, q.Ref, int(q.MuteUntil), "mute")
	if err != nil {
		return output.ChatMuteRow{}, err
	}
	return row, nil
}

// UnmuteChat restores chat notifications.
func UnmuteChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.UnmuteQuery) (output.ChatMuteRow, error) {
	return updateChatMute(ctx, api, resolver, q.Ref, 0, "unmute")
}

func updateChatMute(ctx context.Context, api *tg.Client, resolver *peer.Resolver, target ref.Ref, muteUntil int, action string) (output.ChatMuteRow, error) {
	resolved, err := resolver.Resolve(ctx, target)
	if err != nil {
		return output.ChatMuteRow{}, err
	}
	settings := &tg.InputPeerNotifySettings{}
	settings.SetMuteUntil(muteUntil)
	req := &tg.AccountUpdateNotifySettingsRequest{
		Peer:     &tg.InputNotifyPeer{Peer: resolved.InputPeer},
		Settings: *settings,
	}
	if _, err := api.AccountUpdateNotifySettings(ctx, req); err != nil {
		return output.ChatMuteRow{}, err
	}
	return output.ChatMuteRow{Action: action, Peer: output.PeerRefFromResolved(resolved)}, nil
}

// ReadChat marks a chat as read.
func ReadChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.ReadQuery) error {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return err
	}
	if ch, ok := inputChannelFromPeer(resolved.InputPeer); ok {
		_, err := api.ChannelsReadHistory(ctx, &tg.ChannelsReadHistoryRequest{
			Channel: ch,
			MaxID:   q.MaxID,
		})
		return err
	}
	_, err = api.MessagesReadHistory(ctx, &tg.MessagesReadHistoryRequest{
		Peer:  resolved.InputPeer,
		MaxID: q.MaxID,
	})
	return err
}

// JoinChat joins a channel/supergroup or invite link.
func JoinChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.MembershipQuery) (output.ChatMembershipRow, error) {
	row := output.ChatMembershipRow{Action: "join"}
	if q.Ref.IsInviteLink() {
		if _, err := api.MessagesImportChatInvite(ctx, q.Ref.InviteHash()); err != nil {
			msg := err.Error()
			switch {
			case strings.Contains(msg, "INVITE_HASH_EXPIRED"), strings.Contains(msg, "INVITE_HASH_INVALID"):
				return output.ChatMembershipRow{}, fmt.Errorf("%w: %s", chaterr.ErrInvalidInvite, msg)
			case strings.Contains(msg, "USER_ALREADY_PARTICIPANT"):
				row.AlreadyMember = true
				return fillInviteTarget(ctx, api, q.Ref, row), nil
			}
			return output.ChatMembershipRow{}, err
		}
		return fillInviteTarget(ctx, api, q.Ref, row), nil
	}

	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.ChatMembershipRow{}, err
	}
	ch, ok := resolved.InputPeer.(*tg.InputPeerChannel)
	if !ok {
		return output.ChatMembershipRow{}, fmt.Errorf("%w: only channels / supergroups can be joined by ref", command.ErrUsage)
	}
	inCh := &tg.InputChannel{ChannelID: ch.ChannelID, AccessHash: ch.AccessHash}
	if _, err := api.ChannelsJoinChannel(ctx, inCh); err != nil {
		if strings.Contains(err.Error(), "USER_ALREADY_PARTICIPANT") {
			row.AlreadyMember = true
		} else {
			return output.ChatMembershipRow{}, err
		}
	}
	row.Peer = output.PeerRefFromResolved(resolved)
	row.Role = "member"
	return row, nil
}

// LeaveChat leaves a channel or supergroup.
func LeaveChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.MembershipQuery) (output.ChatMembershipRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.ChatMembershipRow{}, err
	}
	ch, ok := resolved.InputPeer.(*tg.InputPeerChannel)
	if !ok {
		return output.ChatMembershipRow{}, fmt.Errorf("%w: only channels / supergroups can be left by ref", command.ErrUsage)
	}
	inCh := &tg.InputChannel{ChannelID: ch.ChannelID, AccessHash: ch.AccessHash}
	if _, err := api.ChannelsLeaveChannel(ctx, inCh); err != nil {
		if strings.Contains(err.Error(), "USER_NOT_PARTICIPANT") {
			return output.ChatMembershipRow{}, fmt.Errorf("%w: not a member of %s", command.ErrUsage, q.Ref.String())
		}
		return output.ChatMembershipRow{}, err
	}
	return output.ChatMembershipRow{Action: "leave", Peer: output.PeerRefFromResolved(resolved)}, nil
}

// ListChatMembers resolves a channel before participant loading.
func ListChatMembers(ctx context.Context, resolver *peer.Resolver, q actionchat.MembersQuery) ([]output.MemberRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}
	if _, ok := resolved.InputPeer.(*tg.InputPeerChannel); !ok {
		return nil, fmt.Errorf("%w: chat members only supported on channels/supergroups", command.ErrUnsupported)
	}
	return nil, fmt.Errorf("%w: chat members is not available yet", command.ErrUnsupported)
}

func fillInviteTarget(ctx context.Context, api *tg.Client, target ref.Ref, row output.ChatMembershipRow) output.ChatMembershipRow {
	ci, err := api.MessagesCheckChatInvite(ctx, target.InviteHash())
	if err != nil {
		return row
	}
	switch v := ci.(type) {
	case *tg.ChatInviteAlready:
		row.Peer = output.PeerRefFromChat(v.Chat)
	case *tg.ChatInvitePeek:
		row.Peer = output.PeerRefFromChat(v.Chat)
	}
	row.Role = "member"
	return row
}

func inputChannelFromPeer(p tg.InputPeerClass) (tg.InputChannelClass, bool) {
	switch v := p.(type) {
	case *tg.InputPeerChannel:
		return &tg.InputChannel{ChannelID: v.ChannelID, AccessHash: v.AccessHash}, true
	case *tg.InputPeerChannelFromMessage:
		return &tg.InputChannelFromMessage{Peer: v.Peer, MsgID: v.MsgID, ChannelID: v.ChannelID}, true
	}
	return nil, false
}
