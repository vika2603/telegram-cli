package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestCreateChat_PassesFieldsThrough(t *testing.T) {
	row, err := actionchat.CreateChat(context.Background(), actionchat.CreateChatRequest{
		Title: "Team",
		About: "our group",
		Forum: true,
	}, func(_ context.Context, q actionchat.CreateChatQuery) (output.ChatRow, error) {
		require.Equal(t, "Team", q.Title)
		require.Equal(t, "our group", q.About)
		require.False(t, q.Broadcast)
		require.True(t, q.Forum)
		return output.ChatRow{Kind: "chat", Title: q.Title}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "Team", row.Title)
}

func TestCreateChat_RejectsEmptyTitle(t *testing.T) {
	_, err := actionchat.CreateChat(context.Background(), actionchat.CreateChatRequest{}, func(context.Context, actionchat.CreateChatQuery) (output.ChatRow, error) {
		t.Fatal("create must not run with an empty title")
		return output.ChatRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestCreateChat_RejectsOverlongTitle(t *testing.T) {
	_, err := actionchat.CreateChat(context.Background(), actionchat.CreateChatRequest{
		Title: strings.Repeat("x", 129),
	}, func(context.Context, actionchat.CreateChatQuery) (output.ChatRow, error) {
		t.Fatal("create must not run with an overlong title")
		return output.ChatRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestCreateChat_RejectsForumChannel(t *testing.T) {
	_, err := actionchat.CreateChat(context.Background(), actionchat.CreateChatRequest{
		Title:     "News",
		Broadcast: true,
		Forum:     true,
	}, func(context.Context, actionchat.CreateChatQuery) (output.ChatRow, error) {
		t.Fatal("create must not run for a forum broadcast channel")
		return output.ChatRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}
