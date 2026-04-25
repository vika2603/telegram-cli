package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestRenderChatList_EmitsTableWithHeader(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.ChatRow{
		{ID: 1, Kind: "user", Title: "Pavel", Username: "durov"},
		{ID: -100, Kind: "channel", Title: "News", Username: "news"},
	}
	require.NoError(t, output.RenderChatList(ios, rows))
	got := stdout.String()
	require.Contains(t, got, "REF")
	require.Contains(t, got, "KIND")
	require.Contains(t, got, "Pavel")
	require.Contains(t, got, "News")
}

func TestRenderChatList_TTYRendersChatFeed(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	ios.SetStdoutTTY(true)
	rows := []output.ChatRow{
		{
			ID:          -100,
			Ref:         "@news",
			Kind:        "channel",
			Title:       "风向旗参考快讯",
			Username:    "news",
			UnreadCount: 42,
			Last: &output.MessageSummary{
				Text: "日本签署意见在防御中国相关黑客的国际网络安全文件",
			},
		},
	}
	require.NoError(t, output.RenderChatList(ios, rows))
	got := stdout.String()
	require.NotContains(t, got, "REF")
	require.NotContains(t, got, "TYPE")
	require.NotContains(t, got, "TITLE")
	require.NotContains(t, got, "UNREAD")
	require.NotContains(t, got, "LAST")
	require.Contains(t, got, "@news")
	require.Contains(t, got, "风向旗参考快讯")
	require.NotContains(t, got, "风向旗参考快讯 @news")
	require.Contains(t, got, "channel · 42 unread")
	require.Contains(t, got, "  日本签署意见")
}

func TestRenderChatList_TTYDoesNotTruncateLongRefs(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	ios.SetStdoutTTY(true)
	longRef := "@very_very_long_username_that_must_remain_copyable"
	rows := []output.ChatRow{
		{
			ID:       1,
			Ref:      longRef,
			Kind:     "user",
			Title:    "Copy Target",
			Username: "very_very_long_username_that_must_remain_copyable",
			Last:     &output.MessageSummary{Text: "hello"},
		},
	}

	require.NoError(t, output.RenderChatList(ios, rows))
	got := stdout.String()
	require.Contains(t, got, longRef)
	require.NotContains(t, got, "@very_very_long_username_that…")
}

func TestChatMembershipRow_AlreadyMember(t *testing.T) {
	row := output.ChatMembershipRow{
		Action:        "join",
		Peer:          output.PeerRef{ID: 123, Kind: "channel", Title: "Test"},
		AlreadyMember: true,
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteChatMembershipJSON(&buf, row))

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Equal(t, "join", got["action"])
	require.Equal(t, true, got["already_member"])
	_, hasRole := got["role"]
	require.False(t, hasRole, "role should be omitted when empty")
}

func TestChatMembershipRow_WithRole(t *testing.T) {
	row := output.ChatMembershipRow{
		Action: "join",
		Peer:   output.PeerRef{ID: 456, Kind: "channel", Title: "Other", Username: "other"},
		Role:   "member",
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteChatMembershipJSON(&buf, row))

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Equal(t, "join", got["action"])
	require.Equal(t, "member", got["role"])
	_, hasAlready := got["already_member"]
	require.False(t, hasAlready, "already_member should be omitted when false")
	peer, ok := got["peer"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "other", peer["username"])
}

func TestChatMuteRow_Forever(t *testing.T) {
	row := output.ChatMuteRow{
		Action:    "mute",
		Peer:      output.PeerRef{ID: 789, Kind: "channel", Title: "Silent"},
		MuteUntil: "forever",
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteChatMuteJSON(&buf, row))

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Equal(t, "mute", got["action"])
	require.Equal(t, "forever", got["mute_until"])
	peer, ok := got["peer"].(map[string]any)
	require.True(t, ok)
	require.InDelta(t, float64(789), peer["id"], 0)
}

func TestChatMuteRow_OmitsMuteUntilWhenEmpty(t *testing.T) {
	row := output.ChatMuteRow{
		Action: "unmute",
		Peer:   output.PeerRef{ID: 789, Kind: "channel", Title: "Silent"},
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteChatMuteJSON(&buf, row))

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Equal(t, "unmute", got["action"])
	_, hasMuteUntil := got["mute_until"]
	require.False(t, hasMuteUntil, "mute_until should be omitted when empty")
}

func TestChatFolderRow_Archive(t *testing.T) {
	row := output.ChatFolderRow{
		Action: "archive",
		Peer:   output.PeerRef{ID: 123, Kind: "channel", Title: "Test", Username: "test"},
		Folder: 1,
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteChatFolderJSON(&buf, row))

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Equal(t, "archive", got["action"])
	require.InDelta(t, float64(1), got["folder"], 0)
	peer, ok := got["peer"].(map[string]any)
	require.True(t, ok)
	require.InDelta(t, float64(123), peer["id"], 0)
	require.Equal(t, "channel", peer["kind"])
	require.Equal(t, "test", peer["username"])
}

func TestChatFolderRow_Unarchive(t *testing.T) {
	row := output.ChatFolderRow{
		Action: "unarchive",
		Peer:   output.PeerRef{ID: 456, Kind: "chat", Title: "Group"},
		Folder: 0,
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteChatFolderJSON(&buf, row))

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Equal(t, "unarchive", got["action"])
	require.InDelta(t, float64(0), got["folder"], 0)
	peer, ok := got["peer"].(map[string]any)
	require.True(t, ok)
	require.InDelta(t, float64(456), peer["id"], 0)
	_, hasUsername := peer["username"]
	require.False(t, hasUsername, "username should be omitted when empty")
}
