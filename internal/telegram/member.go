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
func InviteToChat(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.InviteQuery) ([]output.InviteRow, error) {
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

	res, err := api.ChannelsInviteToChannel(ctx, &tg.ChannelsInviteToChannelRequest{
		Channel: inCh,
		Users:   inputUsers,
	})
	if err != nil {
		return nil, err
	}

	// Telegram accepts the request even for users it could not actually add
	// (e.g. their privacy settings disallow being added); those land in
	// MissingInvitees rather than producing an error.
	missing := make(map[int64]tg.MissingInvitee, len(res.MissingInvitees))
	for _, m := range res.MissingInvitees {
		missing[m.UserID] = m
	}

	rows := make([]output.InviteRow, 0, len(userResolved))
	for _, r := range userResolved {
		row := output.InviteRow{Peer: output.PeerRefFromResolved(r), Invited: true}
		if m, ok := missing[r.ID]; ok {
			row.Invited = false
			row.SkipReason = inviteSkipReason(m)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// inviteSkipReason classifies why Telegram declined to add a user.
func inviteSkipReason(m tg.MissingInvitee) string {
	switch {
	case m.PremiumWouldAllowInvite:
		return "premium_would_allow_invite"
	case m.PremiumRequiredForPm:
		return "premium_required_for_pm"
	default:
		return "privacy_restricted"
	}
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

	// A broad admin set covering both supergroups and broadcast channels.
	// PostMessages/EditMessages only apply to broadcast channels (ignored by
	// supergroups); the rest apply to supergroups. AddAdmins is deliberately
	// left off so a promoted admin can't mint further admins.
	var rights tg.ChatAdminRights
	if !q.Demote {
		rights = tg.ChatAdminRights{
			ChangeInfo:     true,
			PostMessages:   true,
			EditMessages:   true,
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
