package telegram

import (
	"context"
	"testing"
	"time"

	msgpeer "github.com/gotd/td/telegram/message/peer"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"
)

// dispatchEntities mirrors tg.Entities so we can hand-craft an update
// context with users/chats/channels populated.
func dispatchEntities(users map[int64]*tg.User, chats map[int64]*tg.Chat, channels map[int64]*tg.Channel) tg.Entities {
	return tg.Entities{Users: users, Chats: chats, Channels: channels}
}

func TestRegisterWatchHandlers_NewChannelMessage(t *testing.T) {
	disp := tg.NewUpdateDispatcher()
	events := make(chan WatchEvent, 4)
	RegisterWatchHandlers(disp, WatchFilter{}, nil, events)

	te := dispatchEntities(nil, nil, map[int64]*tg.Channel{
		99: {ID: 99, AccessHash: 990, Title: "Src", Username: "src", Broadcast: true},
	})
	msg := &tg.Message{
		ID:      77,
		Date:    int(time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC).Unix()),
		PeerID:  &tg.PeerChannel{ChannelID: 99},
		Message: "hi",
	}
	err := disp.Handle(context.Background(), &tg.UpdateShort{
		Update: &tg.UpdateNewChannelMessage{Message: msg},
		Date:   msg.Date,
	})
	// UpdateShort.short() zeros out the entities map, so peer.Title is empty
	// in the synthesized row — but the event still fires. Fall back to a
	// proper Updates envelope for the title check.
	require.NoError(t, err)
	select {
	case ev := <-events:
		require.Equal(t, EventNewMessage, ev.Kind)
		require.Equal(t, 77, ev.Row.ID)
		require.Equal(t, "hi", ev.Row.Text)
	case <-time.After(time.Second):
		t.Fatal("expected message event")
	}

	// Now use a full Updates envelope so msgpeer.Entities is populated.
	err = disp.Handle(context.Background(), &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateNewChannelMessage{Message: msg}},
		Chats:   []tg.ChatClass{te.Channels[99]},
	})
	require.NoError(t, err)
	select {
	case ev := <-events:
		require.Equal(t, EventNewMessage, ev.Kind)
		require.Equal(t, "channel", ev.Row.FromKind)
		require.Equal(t, "Src", ev.Row.FromTitle)
		require.Equal(t, "src", ev.Row.FromUsername)
	case <-time.After(time.Second):
		t.Fatal("expected enriched message event")
	}
}

func TestRegisterWatchHandlers_PeerFilterDropsOthers(t *testing.T) {
	disp := tg.NewUpdateDispatcher()
	events := make(chan WatchEvent, 4)
	allowed := int64(-1_000_000_000_099)
	RegisterWatchHandlers(disp, WatchFilter{
		PeerIDs: map[int64]struct{}{allowed: {}},
	}, nil, events)

	allow := &tg.UpdateNewChannelMessage{Message: &tg.Message{
		ID: 1, PeerID: &tg.PeerChannel{ChannelID: 99}, Message: "keep",
	}}
	drop := &tg.UpdateNewChannelMessage{Message: &tg.Message{
		ID: 2, PeerID: &tg.PeerChannel{ChannelID: 42}, Message: "skip",
	}}
	require.NoError(t, disp.Handle(context.Background(), &tg.Updates{
		Updates: []tg.UpdateClass{drop, allow},
		Chats: []tg.ChatClass{
			&tg.Channel{ID: 42, Broadcast: true},
			&tg.Channel{ID: 99, Broadcast: true},
		},
	}))

	got := drainEvents(events)
	require.Len(t, got, 1)
	require.Equal(t, 1, got[0].Row.ID)
}

func TestRegisterWatchHandlers_KindFilter(t *testing.T) {
	disp := tg.NewUpdateDispatcher()
	events := make(chan WatchEvent, 4)
	RegisterWatchHandlers(disp, WatchFilter{
		Kinds: map[WatchEventKind]struct{}{EventEditMessage: {}},
	}, nil, events)

	msg := &tg.Message{ID: 1, PeerID: &tg.PeerChannel{ChannelID: 99}, Message: "x"}
	require.NoError(t, disp.Handle(context.Background(), &tg.Updates{
		Updates: []tg.UpdateClass{
			&tg.UpdateNewChannelMessage{Message: msg},
			&tg.UpdateEditChannelMessage{Message: msg},
		},
		Chats: []tg.ChatClass{&tg.Channel{ID: 99, Broadcast: true}},
	}))

	got := drainEvents(events)
	require.Len(t, got, 1)
	require.Equal(t, EventEditMessage, got[0].Kind)
}

func TestRegisterWatchHandlers_DeleteChannelMessagesBatchesIDs(t *testing.T) {
	disp := tg.NewUpdateDispatcher()
	events := make(chan WatchEvent, 4)
	RegisterWatchHandlers(disp, WatchFilter{}, nil, events)

	require.NoError(t, disp.Handle(context.Background(), &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateDeleteChannelMessages{
			ChannelID: 99,
			Messages:  []int{101, 102, 103},
		}},
	}))

	got := drainEvents(events)
	require.Len(t, got, 1)
	require.Equal(t, EventDeleteMessages, got[0].Kind)
	require.Len(t, got[0].Rows, 3)
	require.Equal(t, 101, got[0].Rows[0].ID)
	require.Equal(t, int64(-1_000_000_000_099), got[0].Rows[0].FromID)
}

func TestRegisterWatchHandlers_ContextCancelStopsEmit(t *testing.T) {
	disp := tg.NewUpdateDispatcher()
	// Unbuffered so the send blocks until ctx cancels.
	events := make(chan WatchEvent)
	RegisterWatchHandlers(disp, WatchFilter{}, nil, events)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := disp.Handle(ctx, &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateNewChannelMessage{Message: &tg.Message{
			ID: 1, PeerID: &tg.PeerChannel{ChannelID: 99}, Message: "x",
		}}},
		Chats: []tg.ChatClass{&tg.Channel{ID: 99, Broadcast: true}},
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestRegisterWatchHandlers_ResolveRefIsCalledForBaseRef(t *testing.T) {
	disp := tg.NewUpdateDispatcher()
	events := make(chan WatchEvent, 4)

	var called bool
	RegisterWatchHandlers(disp, WatchFilter{},
		func(p tg.PeerClass, _ msgpeer.Entities) string {
			called = true
			if _, ok := p.(*tg.PeerChannel); ok {
				return "@src"
			}
			return ""
		}, events)

	require.NoError(t, disp.Handle(context.Background(), &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateNewChannelMessage{Message: &tg.Message{
			ID: 7, PeerID: &tg.PeerChannel{ChannelID: 99}, Message: "x",
		}}},
		Chats: []tg.ChatClass{&tg.Channel{ID: 99, Username: "src", Broadcast: true}},
	}))
	got := drainEvents(events)
	require.Len(t, got, 1)
	require.True(t, called)
	require.Equal(t, "@src:7", got[0].Row.Ref)
}

func TestRegisterWatchHandlers_NilResolveRefDerivesReplyableRef(t *testing.T) {
	disp := tg.NewUpdateDispatcher()
	events := make(chan WatchEvent, 8)
	RegisterWatchHandlers(disp, WatchFilter{}, nil, events)

	// Channel with a username -> @username:msgID.
	require.NoError(t, disp.Handle(context.Background(), &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateNewChannelMessage{Message: &tg.Message{
			ID: 7, PeerID: &tg.PeerChannel{ChannelID: 99}, Message: "x",
		}}},
		Chats: []tg.ChatClass{&tg.Channel{ID: 99, AccessHash: 990, Username: "src", Broadcast: true}},
	}))

	// Private chat with a known user (access hash present) -> u:ID:HASH:msgID.
	require.NoError(t, disp.Handle(context.Background(), &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateNewMessage{Message: &tg.Message{
			ID: 8, PeerID: &tg.PeerUser{UserID: 555}, Message: "y",
		}}},
		Users: []tg.UserClass{&tg.User{ID: 555, AccessHash: 4242}},
	}))

	// Cold channel the update omits -> hash-less ref that still names the peer.
	require.NoError(t, disp.Handle(context.Background(), &tg.Updates{
		Updates: []tg.UpdateClass{&tg.UpdateNewChannelMessage{Message: &tg.Message{
			ID: 9, PeerID: &tg.PeerChannel{ChannelID: 42}, Message: "z",
		}}},
	}))

	got := drainEvents(events)
	require.Len(t, got, 3)
	require.Equal(t, "@src:7", got[0].Row.Ref)
	require.Equal(t, "u:555:4242:8", got[1].Row.Ref)
	require.Equal(t, "c:42:0:9", got[2].Row.Ref)
}

// drainEvents reads everything from ch until quiet for 200ms.
func drainEvents(ch <-chan WatchEvent) []WatchEvent {
	var out []WatchEvent
	for {
		select {
		case ev := <-ch:
			out = append(out, ev)
		case <-time.After(200 * time.Millisecond):
			return out
		}
	}
}
