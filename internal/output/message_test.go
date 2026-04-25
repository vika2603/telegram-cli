package output_test

import (
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
