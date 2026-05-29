package output_test

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestRenderMessages_ShowsIDAndText(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.MessageRow{
		{ID: 10, Ref: "@ada:10", Date: "2026-04-23T12:00:00Z", FromID: 1, FromTitle: "Ada Lovelace", FromUsername: "ada", Text: "hello"},
	}
	require.NoError(t, output.RenderMessages(ios, rows))
	got := stdout.String()
	require.Contains(t, got, "REF")
	require.Contains(t, got, "@ada:10")
	require.Contains(t, got, "Ada Lovelace (@ada)")
	require.Contains(t, got, "hello")
}

func TestRenderMessages_TTYRendersMessageFeed(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	ios.SetStdoutTTY(true)
	rows := []output.MessageRow{
		{
			ID:           10,
			Ref:          "@chat:10",
			Date:         "2026-04-23T12:00:00Z",
			FromID:       1,
			FromTitle:    "Ada Lovelace",
			FromUsername: "ada",
			Text:         "hello from the feed",
			ReplyTo:      &output.ReplyInfo{MessageID: 9},
		},
		{
			ID:        11,
			Ref:       "@chat:11",
			Date:      "2026-04-23T12:01:00Z",
			HasMedia:  true,
			MediaKind: "photo",
			Text:      "caption",
		},
	}
	require.NoError(t, output.RenderMessages(ios, rows))

	got := stdout.String()
	require.NotContains(t, got, "REF")
	require.Contains(t, got, "@chat:10")
	require.Contains(t, got, "Ada Lovelace (@ada)")
	require.Contains(t, got, "reply 9")
	require.Contains(t, got, "  hello from the feed")
	require.Contains(t, got, "@chat:11")
	require.Contains(t, got, "  [photo] caption")
}

func TestRenderMessages_TruncationKeepsValidUTF8(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	long := strings.Repeat("日", 100)
	rows := []output.MessageRow{{ID: 1, Date: "2026-04-23T12:00:00Z", Text: long}}
	require.NoError(t, output.RenderMessages(ios, rows))
	require.True(t, utf8.ValidString(stdout.String()))
}

func TestRenderMessages_ChannelPostBlankFrom(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.MessageRow{{ID: 5, Date: "2026-04-23T12:00:00Z", FromID: 0, Text: "post"}}
	require.NoError(t, output.RenderMessages(ios, rows))
	require.NotContains(t, stdout.String(), "\t0\t")
}

func TestRenderMessages_MediaPreview(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.MessageRow{{ID: 7, Date: "2026-04-23T12:00:00Z", HasMedia: true, MediaKind: "photo"}}
	require.NoError(t, output.RenderMessages(ios, rows))
	require.Contains(t, stdout.String(), "[photo]")
}

func TestRenderMessages_TTYDoesNotTruncateLongRefs(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	ios.SetStdoutTTY(true)
	ref := "c:1234567890123:987654321098765432:77"
	rows := []output.MessageRow{{ID: 77, Ref: ref, Date: "2026-04-23T12:00:00Z", Text: "hello"}}
	require.NoError(t, output.RenderMessages(ios, rows))
	require.Contains(t, stdout.String(), ref)
}

// TestMessageRow_MarshalJSON_ReplyToObject guards the shape change to
// reply_to: was `reply_to: <int>`, now an object with message_id,
// peer_id, forum_topic, top_id, quote_text, and friends.
func TestMessageRow_MarshalJSON_ReplyToObject(t *testing.T) {
	row := output.MessageRow{
		ID:   77,
		Date: "2026-04-23T12:00:00Z",
		Text: "thread comment",
		ReplyTo: &output.ReplyInfo{
			MessageID:     42,
			PeerID:        -1001234567890,
			ForumTopic:    true,
			TopID:         100,
			QuoteText:     "the important bit",
			QuoteEntities: []output.MessageEntity{{Type: "bold", Text: "important"}},
			QuoteOffset:   4,
			QuoteIsManual: true,
		},
	}
	b, err := json.Marshal(row)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))

	require.NotNil(t, got["reply_to"], "reply_to must be present as an object")
	rt := got["reply_to"].(map[string]any)
	require.InDelta(t, 42.0, rt["message_id"], 0)
	require.InDelta(t, -1001234567890.0, rt["peer_id"], 0)
	require.Equal(t, true, rt["forum_topic"])
	require.InDelta(t, 100.0, rt["top_id"], 0)
	require.Equal(t, "the important bit", rt["quote_text"])
	require.InDelta(t, 4.0, rt["quote_offset"], 0)
	require.Equal(t, true, rt["quote_is_manual"])
	require.NotNil(t, rt["quote_entities"])
}

// TestMessageRow_MarshalJSON_ReplyToOmitsWhenAbsent asserts a plain
// message emits no reply_to key (nil pointer + omitempty).
func TestMessageRow_MarshalJSON_ReplyToOmitsWhenAbsent(t *testing.T) {
	row := output.MessageRow{ID: 1, Date: "2026-04-23T12:00:00Z", Text: "hi"}
	b, err := json.Marshal(row)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	require.NotContains(t, got, "reply_to")
}

// TestMessageRow_MarshalJSON_EditDateGroupedReactions guards the
// agent-CLI fields added on top of PR #2's forward/entities/buttons:
// EditDate (cache key for "did this change since last poll?"),
// GroupedID (album discriminator so N-photo posts are not seen as
// N independent messages), and Reactions (engagement signal).
func TestMessageRow_MarshalJSON_EditDateGroupedReactions(t *testing.T) {
	row := output.MessageRow{
		ID:        9,
		Date:      "2026-04-23T12:00:00Z",
		EditDate:  "2026-04-23T13:00:00Z",
		GroupedID: 1234567890,
		Text:      "hi",
		Reactions: []output.ReactionCount{
			{Kind: "emoji", Emoji: "👍", Count: 5, SelfReacted: true},
			{Kind: "emoji", Emoji: "🚀", Count: 2},
			{Kind: "custom_emoji", CustomEmojiID: 555, Count: 1},
		},
	}
	b, err := json.Marshal(row)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))

	require.Equal(t, "2026-04-23T13:00:00Z", got["edit_date"])
	require.InDelta(t, 1234567890.0, got["grouped_id"], 0)
	reactions := got["reactions"].([]any)
	require.Len(t, reactions, 3)
	r0 := reactions[0].(map[string]any)
	require.Equal(t, "emoji", r0["kind"])
	require.Equal(t, "👍", r0["emoji"])
	require.InDelta(t, 5.0, r0["count"], 0)
	require.Equal(t, true, r0["self_reacted"])
	r2 := reactions[2].(map[string]any)
	require.Equal(t, "custom_emoji", r2["kind"])
	require.InDelta(t, 555.0, r2["custom_emoji_id"], 0)
}

// TestMessageRow_MarshalJSON_OmitsEditDateGroupedReactions asserts the
// three new fields are properly omitempty so old shaped rows are byte
// compatible with the pre-feature output.
func TestMessageRow_MarshalJSON_OmitsEditDateGroupedReactions(t *testing.T) {
	row := output.MessageRow{ID: 1, Date: "2026-04-23T12:00:00Z", Text: "hi"}
	b, err := json.Marshal(row)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	require.NotContains(t, got, "edit_date")
	require.NotContains(t, got, "grouped_id")
	require.NotContains(t, got, "reactions")
}

func TestMessageRow_MarshalJSON_EntitiesForwardButtons(t *testing.T) {
	row := output.MessageRow{
		ID:   1,
		Ref:  "@dst:1",
		Date: "2026-04-23T12:00:00Z",
		Text: "hi",
		Entities: []output.MessageEntity{
			{Type: "text_url", Text: "hi", URL: "https://e.com"},
		},
		Buttons: []output.MessageButton{
			{Row: 0, Text: "Open", URL: "https://e.com", Type: "url"},
		},
		Forward: &output.ForwardInfo{
			Date:          "2026-04-22T00:00:00Z",
			ChannelPostID: 7,
			Link:          "https://t.me/src/7",
			From: &output.PeerObject{
				Ref:      "@src",
				Type:     "channel",
				Title:    "Src",
				Username: "src",
				Link:     "https://t.me/src",
			},
		},
	}
	b, err := json.Marshal(row)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))

	require.Contains(t, got, "entities")
	require.Contains(t, got, "buttons")
	require.Contains(t, got, "forward")
	fwd := got["forward"].(map[string]any)
	require.Equal(t, "https://t.me/src/7", fwd["link"])
	require.InDelta(t, 7, fwd["channel_post_id"], 0)
	from := fwd["from"].(map[string]any)
	require.Equal(t, "src", from["username"])
}

func TestMessageRow_MarshalJSON_OmitsEmptyForwardAndEntities(t *testing.T) {
	row := output.MessageRow{ID: 1, Date: "2026-04-23T12:00:00Z", Text: "hi"}
	b, err := json.Marshal(row)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	require.NotContains(t, got, "entities")
	require.NotContains(t, got, "buttons")
	require.NotContains(t, got, "forward")
}

// TestMessageSummaryFromRow_CopiesPR2Fields guards the bug-fix: tg
// inbox's `last` field is a MessageSummary derived from MessageRow via
// this helper. The summary used to drop forward / entities / buttons,
// so a forwarded channel post or a message with formatting silently
// looked like plain text in the inbox preview. Verify all three flow
// through.
func TestMessageSummaryFromRow_CopiesPR2Fields(t *testing.T) {
	row := output.MessageRow{
		ID:   42,
		Ref:  "@chan:42",
		Date: "2026-04-23T12:00:00Z",
		Text: "see this",
		Entities: []output.MessageEntity{
			{Type: "bold", Text: "see"},
		},
		Buttons: []output.MessageButton{
			{Row: 0, Text: "Go", URL: "https://x.com", Type: "url"},
		},
		Forward: &output.ForwardInfo{
			FromName:      "Original",
			ChannelPostID: 7,
			Link:          "https://t.me/src/7",
		},
	}
	got := output.MessageSummaryFromRow(row)
	require.Equal(t, row.Entities, got.Entities)
	require.Equal(t, row.Buttons, got.Buttons)
	require.NotNil(t, got.Forward)
	require.Equal(t, "https://t.me/src/7", got.Forward.Link)
}

// TestMessageSummaryFromRow_OmitsEmpty asserts the empty case stays
// empty — adding the three new fields must not cause spurious keys to
// appear on a bare summary.
func TestMessageSummaryFromRow_OmitsEmpty(t *testing.T) {
	row := output.MessageRow{ID: 1, Date: "2026-04-23T12:00:00Z", Text: "hi"}
	got := output.MessageSummaryFromRow(row)
	require.Nil(t, got.Entities)
	require.Nil(t, got.Buttons)
	require.Nil(t, got.Forward)
}

func TestRenderMessages_TTYShowsForwardLabel(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	ios.SetStdoutTTY(true)
	rows := []output.MessageRow{{
		ID:   10,
		Ref:  "@dst:10",
		Date: "2026-04-23T12:00:00Z",
		Text: "post",
		Forward: &output.ForwardInfo{
			From: &output.PeerObject{Username: "src", Title: "Src"},
		},
	}}
	require.NoError(t, output.RenderMessages(ios, rows))
	require.Contains(t, stdout.String(), "fwd @src")
}
