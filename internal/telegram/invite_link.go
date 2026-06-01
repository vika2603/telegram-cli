package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// CreateInviteLink performs messages.exportChatInvite.
func CreateInviteLink(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.InviteLinkCreateQuery) (output.InviteLinkRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.InviteLinkRow{}, err
	}
	req := &tg.MessagesExportChatInviteRequest{Peer: resolved.InputPeer, RequestNeeded: q.RequestNeeded}
	if q.Title != "" {
		req.SetTitle(q.Title)
	}
	if q.ExpireDate > 0 {
		req.SetExpireDate(q.ExpireDate)
	}
	if q.UsageLimit > 0 {
		req.SetUsageLimit(q.UsageLimit)
	}
	inv, err := api.MessagesExportChatInvite(ctx, req)
	if err != nil {
		return output.InviteLinkRow{}, err
	}
	row := inviteLinkToRow(inv)
	row.Action = "create"
	return row, nil
}

// ListInviteLinks performs messages.getExportedChatInvites (single page).
func ListInviteLinks(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.InviteLinkListQuery) ([]output.InviteLinkRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}
	admin := tg.InputUserClass(&tg.InputUserSelf{})
	if q.Admin.Kind != ref.RefKindInvalid {
		ar, err := resolver.Resolve(ctx, q.Admin)
		if err != nil {
			return nil, err
		}
		iu, ok := inputUserFromPeer(ar.InputPeer)
		if !ok {
			return nil, fmt.Errorf("%w: --admin must be a user", command.ErrUsage)
		}
		admin = iu
	}
	req := &tg.MessagesGetExportedChatInvitesRequest{Peer: resolved.InputPeer, AdminID: admin, Limit: q.Limit}
	if q.Revoked {
		req.Revoked = true
	}
	res, err := api.MessagesGetExportedChatInvites(ctx, req)
	if err != nil {
		return nil, err
	}
	rows := make([]output.InviteLinkRow, 0, len(res.Invites))
	for _, inv := range res.Invites {
		rows = append(rows, inviteLinkToRow(inv))
	}
	return rows, nil
}

// RevokeInviteLink performs messages.editExportedChatInvite with revoked=true.
func RevokeInviteLink(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.InviteLinkQuery) (output.InviteLinkRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.InviteLinkRow{}, err
	}
	res, err := api.MessagesEditExportedChatInvite(ctx, &tg.MessagesEditExportedChatInviteRequest{
		Peer:    resolved.InputPeer,
		Link:    q.Link,
		Revoked: true,
	})
	if err != nil {
		return output.InviteLinkRow{}, err
	}
	row := output.InviteLinkRow{Action: "revoke", Link: q.Link}
	if e, ok := res.(*tg.MessagesExportedChatInvite); ok {
		row = inviteLinkToRow(e.Invite)
		row.Action = "revoke"
	}
	return row, nil
}

// DeleteInviteLink performs messages.deleteExportedChatInvite.
func DeleteInviteLink(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.InviteLinkQuery) (output.InviteLinkRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return output.InviteLinkRow{}, err
	}
	if _, err := api.MessagesDeleteExportedChatInvite(ctx, &tg.MessagesDeleteExportedChatInviteRequest{
		Peer: resolved.InputPeer,
		Link: q.Link,
	}); err != nil {
		return output.InviteLinkRow{}, err
	}
	return output.InviteLinkRow{Action: "delete", Link: q.Link}, nil
}

func inviteLinkToRow(c tg.ExportedChatInviteClass) output.InviteLinkRow {
	e, ok := c.(*tg.ChatInviteExported)
	if !ok {
		return output.InviteLinkRow{}
	}
	row := output.InviteLinkRow{
		Link:          e.Link,
		Revoked:       e.Revoked,
		Permanent:     e.Permanent,
		RequestNeeded: e.RequestNeeded,
		AdminID:       e.AdminID,
		CreatedAt:     fmtUnix(e.Date),
	}
	if v, ok := e.GetTitle(); ok {
		row.Title = v
	}
	if v, ok := e.GetExpireDate(); ok {
		row.ExpireDate = fmtUnix(v)
	}
	if v, ok := e.GetUsageLimit(); ok {
		row.UsageLimit = v
	}
	if v, ok := e.GetUsage(); ok {
		row.Usage = v
	}
	if v, ok := e.GetRequested(); ok {
		row.Requested = v
	}
	return row
}
