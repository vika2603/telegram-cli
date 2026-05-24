package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/account"
	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/telegram"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// registerHandlers binds the daemon's application-level RPC table to
// the live session. The closures capture api / res / acct so each
// handler can serve requests off the daemon's existing MTProto
// connection rather than dialing again.
//
// Method names mirror the cli/<verb>/<subverb>/ package layout so
// adding a new daemon-aware command is a one-liner here plus a small
// closure on the CLI side. Failures translate to "method_failed"
// frames; richer exit-code propagation can follow once we wire
// status.Code over the wire.
func registerHandlers(
	srv *Server,
	acct *account.Account,
	api *tg.Client,
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

	// chat.resolve takes the post-validation ShowQuery for the same
	// reason msg.list does — the client has already normalized the ref.
	// Action structs use struct-field names directly on the wire (no
	// json tags). The musttag linter flags this, but touching the
	// action layer just for daemon serialization is worse than the
	// inline directives below. If we ever expose the socket to
	// untrusted clients, add explicit tags and drop them.
	srv.Register("chat.resolve", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionchat.ShowQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag

			return nil, fmt.Errorf("invalid chat.resolve params: %w", err)
		}
		row, err := telegram.ShowChat(ctx, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(row)
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
		return json.Marshal(rows)
	})

	// msg.list takes the post-validation Query (not ListRequest) so the
	// CLI side has already enforced limit caps and date parsing — the
	// daemon trusts its clients on this. If we ever expose the socket
	// to untrusted clients we should re-normalize here.
	srv.Register("msg.list", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.ListQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag

			return nil, fmt.Errorf("invalid msg.list params: %w", err)
		}
		rows, err := telegram.ListMessages(ctx, api, res, q)
		if err != nil {
			return nil, err
		}
		return json.Marshal(rows)
	})

	// Write handlers (Phase 5). Each accepts the post-validation Query
	// shape produced by actionmessage.Normalize*. The CLI side gates
	// on payload shape (no attachments / no stdin) before routing here
	// — those need byte streams the client owns, which the socket
	// protocol does not carry yet.

	// msg.send: text-only sends. SendQuery.Stdin is reset to nil on
	// the wire (io.Reader is not encodable); the CLI guarantees Stdin
	// is unused on the daemon path.
	srv.Register("msg.send", func(ctx context.Context, params json.RawMessage) (json.RawMessage, error) {
		var q actionmessage.SendQuery
		if err := json.Unmarshal(params, &q); err != nil { //nolint:musttag

			return nil, fmt.Errorf("invalid msg.send params: %w", err)
		}
		q.Stdin = nil // never trust client-supplied io.Reader, even if it
		// somehow round-trips; the gate is upstream too.
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
}
