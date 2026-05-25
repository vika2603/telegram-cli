package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/account"
	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	actionprofile "github.com/vika2603/telegram-cli/internal/action/profile"
	actionsearch "github.com/vika2603/telegram-cli/internal/action/search"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// registerHandlers binds the daemon's application-level RPC table to
// the live session. The closures capture api / pm / res / acct so each
// handler can serve requests off the daemon's existing MTProto
// connection rather than dialing again.
//
// Method names mirror the cli/<verb>/<subverb>/ package layout so
// adding a new daemon-aware command is a one-liner here plus a small
// closure on the CLI side. Failures translate to "method_failed"
// frames; richer exit-code propagation can follow once we wire
// status.Code over the wire.
//
// Action structs use struct-field names directly on the wire (no
// json tags). The musttag linter flags this; touching the action
// layer just for daemon serialization is worse than the per-call
// directives below. If the socket is ever exposed to untrusted
// clients, add explicit tags and drop them.
func registerHandlers(
	srv *Server,
	acct *account.Account,
	api *tg.Client,
	pm *peers.Manager,
	res *peer.Resolver,
) {
	_ = acct // reserved for future handlers (recent peer store)

	srv.Register("me.show", func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
		row, err := telegram.ShowMe(ctx, api)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	// ChatRow / MessageRow / SearchMsgRow have a custom MarshalJSON
	// that builds a user-facing envelope shape but lack a matching
	// UnmarshalJSON. Marshalling them directly on the wire would lose
	// every nested field on the client's round trip. Convert through
	// the *Wire type defs (which strip the MarshalJSON method) so
	// encoding/json falls back to the default field-tag serialization
	// the client's plain Unmarshal can reverse.
	srv.Register("chat.resolve", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionchat.ShowQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid chat.resolve params: %w", err)
		}
		row, err := telegram.ShowChat(ctx, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(output.ChatRowWire(row))
	})

	srv.Register("chat.list", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var p actionchat.ListRequest
		if len(params) > 0 {
			if err := json.Unmarshal(params, &p); err != nil { //nolint:musttag
				return nil, fmt.Errorf("invalid chat.list params: %w", err)
			}
		}
		rows, err := telegram.ListChats(ctx, api, p)
		if err != nil {
			return nil, err
		}
		return json.Marshal(output.ChatRowsToWire(rows))
	})

	srv.Register("msg.list", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.ListQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.list params: %w", err)
		}
		rows, err := telegram.ListMessages(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(output.MessageRowsToWire(rows))
	})

	// ── Phase 5: writes (text-only payloads). File uploads remain on
	// the local path because the socket protocol does not carry bytes
	// the client owns.

	srv.Register("msg.send", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.SendQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.send params: %w", err)
		}
		q.Stdin = nil // never trust an io.Reader from the wire
		rows, err := telegram.SendMessage(ctx, api, res, q, io.Discard)
		if err != nil {
			return nil, err
		}
		return json.Marshal(rows)
	})

	srv.Register("msg.edit", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.EditQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.edit params: %w", err)
		}
		row, err := telegram.EditMessage(ctx, api, res, q, io.Discard)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("msg.forward", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.ForwardQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.forward params: %w", err)
		}
		row, err := telegram.ForwardMessages(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("msg.delete", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.DeleteQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.delete params: %w", err)
		}
		if err := telegram.DeleteMessages(ctx, api, res, q); err != nil {
			return nil, err
		}
		return json.RawMessage("true"), nil
	})

	srv.Register("msg.pin", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.PinQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.pin params: %w", err)
		}
		if err := telegram.PinMessage(ctx, api, res, q); err != nil {
			return nil, err
		}
		return json.RawMessage("true"), nil
	})

	srv.Register("msg.react", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.ReactQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.react params: %w", err)
		}
		row, err := telegram.ReactMessage(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	// ── Phase 6: search / contact / profile.

	srv.Register("search.msg", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionsearch.MessageQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid search.msg params: %w", err)
		}
		rows, err := telegram.SearchMessages(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(output.SearchMsgRowsToWire(rows))
	})

	srv.Register("search.chat", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionsearch.ChatQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid search.chat params: %w", err)
		}
		rows, err := telegram.SearchChats(ctx, api, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(rows)
	})

	srv.Register("contact.list", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actioncontact.ListQuery
		if len(params) > 0 {
			if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
				return nil, fmt.Errorf("invalid contact.list params: %w", err)
			}
		}
		rows, err := telegram.ListContacts(ctx, api, pm, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(rows)
	})

	srv.Register("contact.add", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actioncontact.AddQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid contact.add params: %w", err)
		}
		row, err := telegram.AddContact(ctx, api, pm, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("contact.delete", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actioncontact.PeerQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid contact.delete params: %w", err)
		}
		if err := telegram.DeleteContact(ctx, api, res, q); err != nil {
			return nil, err
		}
		return json.RawMessage("true"), nil
	})

	srv.Register("contact.block", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actioncontact.PeerQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid contact.block params: %w", err)
		}
		if err := telegram.BlockContact(ctx, api, res, q); err != nil {
			return nil, err
		}
		return json.RawMessage("true"), nil
	})

	srv.Register("contact.unblock", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actioncontact.PeerQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid contact.unblock params: %w", err)
		}
		if err := telegram.UnblockContact(ctx, api, res, q); err != nil {
			return nil, err
		}
		return json.RawMessage("true"), nil
	})

	srv.Register("profile.set_name", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionprofile.SetNameRequest
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid profile.set_name params: %w", err)
		}
		row, err := telegram.UpdateProfileName(ctx, api, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("profile.set_bio", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionprofile.SetBioRequest
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid profile.set_bio params: %w", err)
		}
		row, err := telegram.UpdateProfileBio(ctx, api, q.Bio)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("profile.set_username", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(params, &q); err != nil {
			return nil, fmt.Errorf("invalid profile.set_username params: %w", err)
		}
		row, err := telegram.UpdateProfileUsername(ctx, api, q.Username)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("profile.set_status", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q struct {
			Offline bool `json:"offline"`
		}
		if err := json.Unmarshal(params, &q); err != nil {
			return nil, fmt.Errorf("invalid profile.set_status params: %w", err)
		}
		row, err := telegram.UpdateProfileStatus(ctx, api, q.Offline)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("profile.delete_photo", func(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
		if err := telegram.DeleteProfilePhoto(ctx, api); err != nil {
			return nil, err
		}
		return json.RawMessage("true"), nil
	})

	// ── Phase 8: remaining chat + message commands.

	srv.Register("chat.join", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionchat.MembershipQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid chat.join params: %w", err)
		}
		row, err := telegram.JoinChat(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("chat.leave", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionchat.MembershipQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid chat.leave params: %w", err)
		}
		row, err := telegram.LeaveChat(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("chat.mark_read", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionchat.ReadQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid chat.mark_read params: %w", err)
		}
		if err := telegram.ReadChat(ctx, api, res, q); err != nil {
			return nil, err
		}
		return json.RawMessage("true"), nil
	})

	srv.Register("chat.mute", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionchat.MuteQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid chat.mute params: %w", err)
		}
		row, err := telegram.MuteChat(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("chat.unmute", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionchat.UnmuteQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid chat.unmute params: %w", err)
		}
		row, err := telegram.UnmuteChat(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	// chat.folder covers both archive and unarchive — FolderQuery.Archived
	// discriminates direction.
	srv.Register("chat.folder", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionchat.FolderQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid chat.folder params: %w", err)
		}
		row, err := telegram.MoveChatToFolder(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
	})

	srv.Register("msg.link", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.LinkQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.link params: %w", err)
		}
		linkPeer, err := telegram.ResolveMessageLinkPeer(ctx, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(linkPeer) //nolint:musttag
	})

	srv.Register("msg.schedule_list", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.ScheduledListQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.schedule_list params: %w", err)
		}
		rows, err := telegram.ListScheduledMessages(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(rows)
	})

	srv.Register("msg.schedule_cancel", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.CancelScheduledQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag
			return nil, fmt.Errorf("invalid msg.schedule_cancel params: %w", err)
		}
		if err := telegram.CancelScheduledMessages(ctx, api, res, q); err != nil {
			return nil, err
		}
		return json.RawMessage("true"), nil
	})
}
