package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// ignoreNotModified treats a *_NOT_MODIFIED RPC error as success: the requested
// rights already match the current state, so the operation is idempotent.
func ignoreNotModified(err error) error {
	if err != nil && strings.Contains(err.Error(), "NOT_MODIFIED") {
		return nil
	}
	return err
}

// applyRightKeys sets the ChatBannedRights bits for each keyword: deny -> true
// (revoked), allow -> false (granted). Allow is applied after deny so it wins
// on overlap.
func applyRightKeys(b *tg.ChatBannedRights, allow, deny []string) {
	set := func(keys []string, v bool) {
		for _, k := range keys {
			switch k {
			case "send":
				b.SendMessages, b.SendPlain = v, v
			case "media":
				b.SendMedia, b.SendPhotos, b.SendVideos, b.SendDocs = v, v, v, v
				b.SendAudios, b.SendVoices, b.SendRoundvideos = v, v, v
			case "stickers":
				b.SendStickers, b.SendGifs = v, v
			case "bots":
				b.SendGames, b.SendInline = v, v
			case "polls":
				b.SendPolls = v
			case "links":
				b.EmbedLinks = v
			case "invite":
				b.InviteUsers = v
			case "pin":
				b.PinMessages = v
			case "info":
				b.ChangeInfo = v
			case "topics":
				b.ManageTopics = v
			}
		}
	}
	set(deny, true)
	set(allow, false)
}

// deniedRightKeys reverses applyRightKeys for display: keywords whose primary
// bit is currently revoked.
func deniedRightKeys(b tg.ChatBannedRights) []string {
	var out []string
	add := func(cond bool, k string) {
		if cond {
			out = append(out, k)
		}
	}
	add(b.SendMessages, "send")
	add(b.SendMedia, "media")
	add(b.SendStickers, "stickers")
	add(b.SendInline, "bots")
	add(b.SendPolls, "polls")
	add(b.EmbedLinks, "links")
	add(b.InviteUsers, "invite")
	add(b.PinMessages, "pin")
	add(b.ChangeInfo, "info")
	add(b.ManageTopics, "topics")
	return out
}

// SetMemberPerms applies per-user permissions via channels.editBanned. It
// starts from the user's current banned rights so --allow/--deny are
// incremental; Unset clears all restrictions.
func SetMemberPerms(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.SetPermsQuery) (output.RightsRow, error) {
	groupResolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.RightsRow{}, err
	}
	inCh, ok := inputChannelFromPeer(groupResolved.InputPeer)
	if !ok {
		return output.RightsRow{}, fmt.Errorf("%w: member permissions are only supported in supergroups", command.ErrUnsupported)
	}
	userResolved, err := resolver.Resolve(ctx, q.User)
	if err != nil {
		return output.RightsRow{}, err
	}
	pr := output.PeerRefFromResolved(userResolved)

	if q.Unset {
		if _, err := api.ChannelsEditBanned(ctx, &tg.ChannelsEditBannedRequest{
			Channel:      inCh,
			Participant:  userResolved.InputPeer,
			BannedRights: tg.ChatBannedRights{},
		}); ignoreNotModified(err) != nil {
			return output.RightsRow{}, err
		}
		return output.RightsRow{Action: "unset-perms", Peer: &pr}, nil
	}

	// Start from the user's current banned rights (incremental).
	base := tg.ChatBannedRights{}
	if got, err := api.ChannelsGetParticipant(ctx, &tg.ChannelsGetParticipantRequest{
		Channel:     inCh,
		Participant: userResolved.InputPeer,
	}); err == nil {
		if banned, ok := got.Participant.(*tg.ChannelParticipantBanned); ok {
			base = banned.BannedRights
		}
	}
	applyRightKeys(&base, q.Allow, q.Deny)
	base.UntilDate = q.UntilDate
	if _, err := api.ChannelsEditBanned(ctx, &tg.ChannelsEditBannedRequest{
		Channel:      inCh,
		Participant:  userResolved.InputPeer,
		BannedRights: base,
	}); ignoreNotModified(err) != nil {
		return output.RightsRow{}, err
	}
	row := output.RightsRow{Action: "set-perms", Peer: &pr, Denied: deniedRightKeys(base)}
	if q.UntilDate > 0 {
		row.Until = fmtUnix(q.UntilDate)
	}
	return row, nil
}

// SetDefaultPerms sets the group's default member permissions via
// messages.editChatDefaultBannedRights, starting from the current defaults.
func SetDefaultPerms(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.PermsQuery) (output.RightsRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.RightsRow{}, err
	}
	inCh, ok := inputChannelFromPeer(resolved.InputPeer)
	if !ok {
		return output.RightsRow{}, fmt.Errorf("%w: default permissions are only supported in supergroups", command.ErrUnsupported)
	}
	base := tg.ChatBannedRights{}
	if full, err := api.ChannelsGetFullChannel(ctx, inCh); err == nil {
		var channelID int64
		if ip, ok := resolved.InputPeer.(*tg.InputPeerChannel); ok {
			channelID = ip.ChannelID
		}
		for _, c := range full.Chats {
			if ch, ok := c.(*tg.Channel); ok && ch.ID == channelID {
				if dbr, ok := ch.GetDefaultBannedRights(); ok {
					base = dbr
				}
				break
			}
		}
	}
	applyRightKeys(&base, q.Allow, q.Deny)
	if _, err := api.MessagesEditChatDefaultBannedRights(ctx, &tg.MessagesEditChatDefaultBannedRightsRequest{
		Peer:         resolved.InputPeer,
		BannedRights: base,
	}); ignoreNotModified(err) != nil {
		return output.RightsRow{}, err
	}
	return output.RightsRow{Action: "perms", Denied: deniedRightKeys(base)}, nil
}
