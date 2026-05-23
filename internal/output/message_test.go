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
			ReplyToID:    9,
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
	require.Equal(t, float64(7), fwd["channel_post_id"])
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
