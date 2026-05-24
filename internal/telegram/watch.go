package telegram

import (
	"context"

	msgpeer "github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/output"
)

// WatchEventKind classifies a real-time update surfaced by Watch.
type WatchEventKind string

const (
	// EventNewMessage covers tg.UpdateNewMessage (private + group) and
	// tg.UpdateNewChannelMessage (channel post). Carries a fully populated
	// MessageRow including forward / entities / buttons.
	EventNewMessage WatchEventKind = "message"
	// EventEditMessage covers tg.UpdateEditMessage and
	// tg.UpdateEditChannelMessage. Same shape as EventNewMessage.
	EventEditMessage WatchEventKind = "edit"
	// EventDeleteMessages reports message IDs the server deleted. Row holds
	// only ID and Ref; other fields are zero.
	EventDeleteMessages WatchEventKind = "delete"
)

// WatchEvent is one ndjson record emitted by `tg watch`.
type WatchEvent struct {
	Kind WatchEventKind    `json:"kind"`
	Row  output.MessageRow `json:"row,omitempty"`
	// Rows is populated for delete events, which batch multiple IDs from a
	// single update. For new/edit events Row is set and Rows is empty.
	Rows []output.MessageRow `json:"rows,omitempty"`
}

// WatchFilter narrows which updates Watch surfaces.
type WatchFilter struct {
	// PeerIDs, when non-empty, restricts events to messages whose owning
	// peer has one of the listed normalized IDs (see peerID in messages.go).
	// Empty means "all dialogs the account participates in".
	PeerIDs map[int64]struct{}
	// Kinds, when non-empty, restricts events by EventKind. Empty means
	// surface every kind RegisterWatchHandlers wires up.
	Kinds map[WatchEventKind]struct{}
}

// RegisterWatchHandlers wires update handlers onto disp that forward
// matching events into out. The caller owns disp (typically passed via
// session.Options.UpdateHandler), and the caller is responsible for
// reading from out until ctx is cancelled. Buffered channels are
// recommended so a slow consumer does not block gotd's dispatcher.
//
// resolveRef builds the baseRef passed to messageToRow so message refs
// look like @username:NNN when possible. A nil resolveRef falls back to
// an empty baseRef, in which case messageToRow uses numeric IDs.
func RegisterWatchHandlers(
	disp tg.UpdateDispatcher,
	filter WatchFilter,
	resolveRef func(p tg.PeerClass, e msgpeer.Entities) string,
	out chan<- WatchEvent,
) {
	if resolveRef == nil {
		resolveRef = func(tg.PeerClass, msgpeer.Entities) string { return "" }
	}

	emit := func(ctx context.Context, ev WatchEvent) error {
		if !filter.acceptKind(ev.Kind) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case out <- ev:
			return nil
		}
	}

	handleMessage := func(kind WatchEventKind) func(ctx context.Context, e tg.Entities, msg tg.MessageClass) error {
		return func(ctx context.Context, e tg.Entities, msg tg.MessageClass) error {
			m, ok := msg.(*tg.Message)
			if !ok {
				return nil
			}
			ents := msgpeer.EntitiesFromUpdate(e)
			if peerID := normalizePeerID(m.PeerID); !filter.acceptPeer(peerID) {
				return nil
			}
			baseRef := resolveRef(m.PeerID, ents)
			row := messageToRow(m, ents, baseRef)
			return emit(ctx, WatchEvent{Kind: kind, Row: row})
		}
	}

	disp.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		return handleMessage(EventNewMessage)(ctx, e, u.Message)
	})
	disp.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		return handleMessage(EventNewMessage)(ctx, e, u.Message)
	})
	disp.OnEditMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateEditMessage) error {
		return handleMessage(EventEditMessage)(ctx, e, u.Message)
	})
	disp.OnEditChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateEditChannelMessage) error {
		return handleMessage(EventEditMessage)(ctx, e, u.Message)
	})
	disp.OnDeleteChannelMessages(func(ctx context.Context, e tg.Entities, u *tg.UpdateDeleteChannelMessages) error {
		// UpdateDeleteChannelMessages does not give us the original peer
		// ref; emit numeric channel ID so consumers can correlate.
		channelID := -1_000_000_000_000 - u.ChannelID
		if !filter.acceptPeer(channelID) {
			return nil
		}
		rows := make([]output.MessageRow, 0, len(u.Messages))
		for _, id := range u.Messages {
			rows = append(rows, output.MessageRow{ID: id, FromID: channelID})
		}
		return emit(ctx, WatchEvent{Kind: EventDeleteMessages, Rows: rows})
	})
	disp.OnDeleteMessages(func(ctx context.Context, _ tg.Entities, u *tg.UpdateDeleteMessages) error {
		// Non-channel deletes do not carry a peer; surface anyway with a
		// zero FromID so unfiltered consumers see them. Peer filter, if
		// set, would have rejected channel-scoped messages above; here we
		// only emit when the filter is empty.
		if len(filter.PeerIDs) > 0 {
			return nil
		}
		rows := make([]output.MessageRow, 0, len(u.Messages))
		for _, id := range u.Messages {
			rows = append(rows, output.MessageRow{ID: id})
		}
		return emit(ctx, WatchEvent{Kind: EventDeleteMessages, Rows: rows})
	})
}

func (f WatchFilter) acceptKind(k WatchEventKind) bool {
	if len(f.Kinds) == 0 {
		return true
	}
	_, ok := f.Kinds[k]
	return ok
}

func (f WatchFilter) acceptPeer(id int64) bool {
	if len(f.PeerIDs) == 0 {
		return true
	}
	_, ok := f.PeerIDs[id]
	return ok
}

// normalizePeerID mirrors peerID() in messages.go so dispatcher filtering
// uses the same scheme MessageRow.FromID exposes.
func normalizePeerID(p tg.PeerClass) int64 {
	if p == nil {
		return 0
	}
	return peerID(p)
}
