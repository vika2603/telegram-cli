package output_test

import (
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
