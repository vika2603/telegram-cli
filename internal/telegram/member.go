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
// channels.editAdmin. Promote grants either the keywords in q.Rights or, when
// none are given, a broad default admin set; demote clears all admin rights.
func SetMemberAdmin(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.PromoteQuery) (output.RightsRow, error) {
	groupResolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.RightsRow{}, err
	}
	inCh, ok := inputChannelFromPeer(groupResolved.InputPeer)
	if !ok {
		return output.RightsRow{}, fmt.Errorf("%w: promote is only supported on channels/supergroups", command.ErrUnsupported)
	}

	userResolved, err := resolver.Resolve(ctx, q.User)
	if err != nil {
		return output.RightsRow{}, err
	}
	inUser, ok := inputUserFromPeer(userResolved.InputPeer)
	if !ok {
		return output.RightsRow{}, fmt.Errorf("%w: %s is not a user", command.ErrUsage, q.User.String())
	}

	var rights tg.ChatAdminRights
	if !q.Demote {
		if len(q.Rights) > 0 {
			rights = adminRightsFromKeys(q.Rights)
		} else {
			rights = defaultAdminRights()
		}
	}

	// editAdmin always overwrites the rank, so when --title was not given on a
	// promote, read the member's current rank and re-send it to avoid silently
	// clearing an existing title.
	rank := q.Title
	if !q.Demote && !q.SetTitle {
		rank = currentAdminRank(ctx, api, inCh, userResolved.InputPeer)
	}

	if _, err := api.ChannelsEditAdmin(ctx, &tg.ChannelsEditAdminRequest{
		Channel:     inCh,
		UserID:      inUser,
		AdminRights: rights,
		Rank:        rank,
	}); err != nil {
		return output.RightsRow{}, err
	}
	pr := output.PeerRefFromResolved(userResolved)
	row := output.RightsRow{Action: "promote", Peer: &pr}
	if q.Demote {
		row.Action = "demote"
	} else {
		row.Granted = grantedAdminKeys(rights)
	}
	return row, nil
}

// adminKeyOrder is the canonical keyword order used when reporting granted
// admin rights, matching the ChatAdminRights field layout.
var adminKeyOrder = []string{
	"info", "post", "edit", "delete", "ban", "invite", "pin",
	"add_admins", "anonymous", "call", "topics",
	"post_stories", "edit_stories", "delete_stories",
}

// defaultAdminRights is the broad admin set granted by `promote` when no
// explicit --rights keywords are passed. PostMessages/EditMessages only apply
// to broadcast channels (ignored by supergroups); the rest apply to
// supergroups. AddAdmins is deliberately left off so a promoted admin can't
// mint further admins.
func defaultAdminRights() tg.ChatAdminRights {
	return tg.ChatAdminRights{
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

// adminRightsFromKeys builds a ChatAdminRights mask from the validated
// keywords accepted by `promote --rights`.
func adminRightsFromKeys(keys []string) tg.ChatAdminRights {
	var r tg.ChatAdminRights
	for _, k := range keys {
		switch k {
		case "info":
			r.ChangeInfo = true
		case "post":
			r.PostMessages = true
		case "edit":
			r.EditMessages = true
		case "delete":
			r.DeleteMessages = true
		case "ban":
			r.BanUsers = true
		case "invite":
			r.InviteUsers = true
		case "pin":
			r.PinMessages = true
		case "add_admins":
			r.AddAdmins = true
		case "anonymous":
			r.Anonymous = true
		case "call":
			r.ManageCall = true
		case "topics":
			r.ManageTopics = true
		case "post_stories":
			r.PostStories = true
		case "edit_stories":
			r.EditStories = true
		case "delete_stories":
			r.DeleteStories = true
		}
	}
	return r
}

// grantedAdminKeys reverses adminRightsFromKeys for display, in canonical order.
func grantedAdminKeys(r tg.ChatAdminRights) []string {
	set := map[string]bool{
		"info":           r.ChangeInfo,
		"post":           r.PostMessages,
		"edit":           r.EditMessages,
		"delete":         r.DeleteMessages,
		"ban":            r.BanUsers,
		"invite":         r.InviteUsers,
		"pin":            r.PinMessages,
		"add_admins":     r.AddAdmins,
		"anonymous":      r.Anonymous,
		"call":           r.ManageCall,
		"topics":         r.ManageTopics,
		"post_stories":   r.PostStories,
		"edit_stories":   r.EditStories,
		"delete_stories": r.DeleteStories,
	}
	var out []string
	for _, k := range adminKeyOrder {
		if set[k] {
			out = append(out, k)
		}
	}
	return out
}

// currentAdminRank returns the participant's existing admin rank (title), or
// "" if they are not currently an admin/creator or the lookup fails.
func currentAdminRank(ctx context.Context, api *tg.Client, ch tg.InputChannelClass, participant tg.InputPeerClass) string {
	got, err := api.ChannelsGetParticipant(ctx, &tg.ChannelsGetParticipantRequest{
		Channel:     ch,
		Participant: participant,
	})
	if err != nil {
		return ""
	}
	switch p := got.Participant.(type) {
	case *tg.ChannelParticipantAdmin:
		return p.Rank
	case *tg.ChannelParticipantCreator:
		return p.Rank
	}
	return ""
}
