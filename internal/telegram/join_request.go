package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// ListJoinRequests performs messages.getChatInviteImporters with requested=true
// (pending join requests). Maps each importer to a MemberRow with role
// "requested".
func ListJoinRequests(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.JoinListQuery) ([]output.MemberRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	// offset_user is a required field; the first page uses InputUserEmpty.
	req := &tg.MessagesGetChatInviteImportersRequest{
		Peer:       resolved.InputPeer,
		Requested:  true,
		Limit:      limit,
		OffsetUser: &tg.InputUserEmpty{},
	}
	if q.Link != "" {
		req.SetLink(q.Link)
	}
	res, err := api.MessagesGetChatInviteImporters(ctx, req)
	if err != nil {
		return nil, err
	}
	users := make(map[int64]*tg.User, len(res.Users))
	for _, uc := range res.Users {
		if u, ok := uc.(*tg.User); ok {
			users[u.ID] = u
		}
	}
	rows := make([]output.MemberRow, 0, len(res.Importers))
	for _, imp := range res.Importers {
		row := output.MemberRow{UserID: imp.UserID, Role: "requested", JoinedAt: fmtUnix(imp.Date)}
		if u, ok := users[imp.UserID]; ok {
			row.FirstName = u.FirstName
			row.LastName = u.LastName
			row.Username = u.Username
			row.IsBot = u.Bot
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// DecideJoinRequest approves or rejects join requests. For --all it calls
// hideAllChatJoinRequests once; otherwise it calls hideChatJoinRequest per
// user, collecting a row each (failures are recorded, not aborted).
func DecideJoinRequest(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionchat.JoinDecisionQuery) ([]output.JoinResultRow, error) {
	resolved, err := resolver.Resolve(ctx, q.Ref)
	if err != nil {
		return nil, err
	}
	action := "approve"
	if !q.Approved {
		action = "deny"
	}

	if q.All {
		req := &tg.MessagesHideAllChatJoinRequestsRequest{Peer: resolved.InputPeer, Approved: q.Approved}
		if q.Link != "" {
			req.SetLink(q.Link)
		}
		if _, err := api.MessagesHideAllChatJoinRequests(ctx, req); err != nil {
			return nil, err
		}
		return []output.JoinResultRow{{Action: action, All: true}}, nil
	}

	rows := make([]output.JoinResultRow, 0, len(q.Users))
	for _, uref := range q.Users {
		ur, err := resolver.Resolve(ctx, uref)
		if err != nil {
			rows = append(rows, output.JoinResultRow{Action: action, Error: err.Error()})
			continue
		}
		pr := output.PeerRefFromResolved(ur)
		iu, ok := inputUserFromPeer(ur.InputPeer)
		if !ok {
			rows = append(rows, output.JoinResultRow{Action: action, Peer: &pr, Error: uref.String() + " is not a user"})
			continue
		}
		_, err = api.MessagesHideChatJoinRequest(ctx, &tg.MessagesHideChatJoinRequestRequest{
			Peer:     resolved.InputPeer,
			UserID:   iu,
			Approved: q.Approved,
		})
		row := output.JoinResultRow{Action: action, Peer: &pr}
		if err != nil {
			row.Error = err.Error()
		}
		rows = append(rows, row)
	}
	return rows, nil
}
