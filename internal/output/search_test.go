package output_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestRenderSearchMsg_ShowsChat(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.SearchMsgRow{
		{MessageID: 7, ChatID: -100, ChatTitle: "News", Date: "2026-04-23T12:00:00Z", Text: "hi"},
	}
	require.NoError(t, output.RenderSearchMsg(ios, rows))
	got := stdout.String()
	require.Contains(t, got, "News")
	require.Contains(t, got, "hi")
}

func TestRenderSearchChat_ShowsSource(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.SearchChatRow{
		{ChatRow: output.ChatRow{ID: 1, Kind: "user", Title: "Alice"}, Source: "my"},
		{ChatRow: output.ChatRow{ID: 2, Kind: "channel", Title: "Public"}, Source: "public"},
	}
	require.NoError(t, output.RenderSearchChat(ios, rows))
	got := stdout.String()
	require.Contains(t, got, "SOURCE")
	require.Contains(t, got, "my")
	require.Contains(t, got, "public")
}

// TestSearchMsgRow_MarshalJSON_SurfacesPR2Fields guards the bug-fix:
// `tg search msg --json` used to lose forward / entities / buttons /
// reply_to / pinned / views / sender details because the row was built
// independently of messageToRow. The row now carries those fields and
// MarshalJSON must round-trip them through the default struct tags.
func TestSearchMsgRow_MarshalJSON_SurfacesPR2Fields(t *testing.T) {
	row := output.SearchMsgRow{
		MessageID:    99,
		Date:         "2026-04-23T12:00:00Z",
		FromID:       1234,
		FromUsername: "alice",
		ChatID:       -100,
		ChatTitle:    "News",
		Text:         "see attached",
		ReplyTo:      &output.ReplyInfo{MessageID: 77},
		Entities: []output.MessageEntity{
			{Type: "bold", Text: "see"},
		},
		Buttons: []output.MessageButton{
			{Row: 0, Text: "open", URL: "https://x.com", Type: "url"},
		},
		Forward: &output.ForwardInfo{
			FromName:      "Channel Alice",
			ChannelPostID: 12,
			Link:          "https://t.me/alice/12",
		},
		HasMedia:  true,
		MediaKind: "photo",
		Views:     500,
		IsPinned:  true,
	}
	raw, err := json.Marshal(row)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))

	require.InDelta(t, 99.0, got["message_id"], 0)
	require.NotNil(t, got["reply_to"])
	require.InDelta(t, 77.0, got["reply_to"].(map[string]any)["message_id"], 0)
	require.InDelta(t, 500.0, got["views"], 0)
	require.Equal(t, true, got["is_pinned"])
	require.Equal(t, true, got["has_media"])
	require.Equal(t, "alice", got["from_username"])
	require.Equal(t, "News", got["chat_title"])
	require.NotNil(t, got["entities"])
	require.NotNil(t, got["buttons"])
	require.NotNil(t, got["forward"])
	fwd := got["forward"].(map[string]any)
	require.Equal(t, "https://t.me/alice/12", fwd["link"])
}

// TestSearchMsgRow_MarshalJSON_OmitsEmpty asserts that adding all those
// fields did not regress the empty-row case: a minimal row still only
// emits the always-required keys.
func TestSearchMsgRow_MarshalJSON_OmitsEmpty(t *testing.T) {
	row := output.SearchMsgRow{MessageID: 5, ChatID: -100, Date: "2026-04-23T12:00:00Z"}
	raw, err := json.Marshal(row)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))

	for _, k := range []string{"entities", "buttons", "forward", "reply_to",
		"views", "is_pinned", "has_media", "from_username"} {
		_, present := got[k]
		require.False(t, present, "key %q should be omitted on empty row", k)
	}
}
