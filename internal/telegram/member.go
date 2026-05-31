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

// InviteToChat adds users to a channel/supergroup via channels.inviteToChannel.
func InviteToChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.InviteQuery) ([]output.PeerRef, error) {
	groupResolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}
	inCh, ok := inputChannelFromPeer(groupResolved.InputPeer)
	if !ok {
		return nil, fmt.Errorf("%w: invite is only supported on channels/supergroups", command.ErrUnsupported)
	}

	inputUsers := make([]tg.InputUserClass, 0, len(q.Users))
	userResolved := make([]peer.Resolved, 0, len(q.Users))
	for _, uRef := range q.Users {
		res, err := resolver.Resolve(ctx, uRef)
		if err != nil {
			return nil, err
		}
		iu, ok := inputUserFromPeer(res.InputPeer)
		if !ok {
			return nil, fmt.Errorf("%w: %s is not a user", command.ErrUsage, uRef.String())
		}
		inputUsers = append(inputUsers, iu)
		userResolved = append(userResolved, res)
	}

	if _, err := api.ChannelsInviteToChannel(ctx, &tg.ChannelsInviteToChannelRequest{
		Channel: inCh,
		Users:   inputUsers,
	}); err != nil {
		return nil, err
	}

	refs := make([]output.PeerRef, 0, len(userResolved))
	for _, r := range userResolved {
		refs = append(refs, output.PeerRefFromResolved(r))
	}
	return refs, nil
}

// SetMemberBanned bans or unbans a user in a channel/supergroup via
// channels.editBanned. Ban sets ViewMessages=true (blocks all access);
// unban clears all restrictions.
func SetMemberBanned(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.BanQuery) (output.PeerRef, error) {
	groupResolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.PeerRef{}, err
	}
	inCh, ok := inputChannelFromPeer(groupResolved.InputPeer)
	if !ok {
		return output.PeerRef{}, fmt.Errorf("%w: ban is only supported on channels/supergroups", command.ErrUnsupported)
	}

	userResolved, err := resolver.Resolve(ctx, q.User)
	if err != nil {
		return output.PeerRef{}, err
	}

	var rights tg.ChatBannedRights
	if !q.Unban {
		rights.ViewMessages = true
	}

	if _, err := api.ChannelsEditBanned(ctx, &tg.ChannelsEditBannedRequest{
		Channel:      inCh,
		Participant:  userResolved.InputPeer,
		BannedRights: rights,
	}); err != nil {
		return output.PeerRef{}, err
	}
	return output.PeerRefFromResolved(userResolved), nil
}

// SetMemberAdmin promotes or demotes a user in a channel/supergroup via
// channels.editAdmin. Promote grants a standard admin rights set; demote
// clears all admin rights.
func SetMemberAdmin(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.PromoteQuery) (output.PeerRef, error) {
	groupResolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.PeerRef{}, err
	}
	inCh, ok := inputChannelFromPeer(groupResolved.InputPeer)
	if !ok {
		return output.PeerRef{}, fmt.Errorf("%w: promote is only supported on channels/supergroups", command.ErrUnsupported)
	}

	userResolved, err := resolver.Resolve(ctx, q.User)
	if err != nil {
		return output.PeerRef{}, err
	}
	inUser, ok := inputUserFromPeer(userResolved.InputPeer)
	if !ok {
		return output.PeerRef{}, fmt.Errorf("%w: %s is not a user", command.ErrUsage, q.User.String())
	}

	var rights tg.ChatAdminRights
	if !q.Demote {
		rights = tg.ChatAdminRights{
			ChangeInfo:     true,
			DeleteMessages: true,
			BanUsers:       true,
			InviteUsers:    true,
			PinMessages:    true,
			ManageCall:     true,
			ManageTopics:   true,
		}
	}

	if _, err := api.ChannelsEditAdmin(ctx, &tg.ChannelsEditAdminRequest{
		Channel:     inCh,
		UserID:      inUser,
		AdminRights: rights,
		Rank:        "",
	}); err != nil {
		return output.PeerRef{}, err
	}
	return output.PeerRefFromResolved(userResolved), nil
}
