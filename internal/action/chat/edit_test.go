package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestEditChat_PassesOnlySetFields(t *testing.T) {
	title := "New Name"
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef: "@grp",
		Title:  &title,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.NotNil(t, q.Title)
		require.Equal(t, "New Name", *q.Title)
		require.Nil(t, q.About, "about not passed -> stays nil")
		return output.ChatRow{Title: *q.Title}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_RejectsNoChange(t *testing.T) {
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef: "@grp",
	}, func(context.Context, actionchat.EditChatQuery) (output.ChatRow, error) {
		t.Fatal("edit must not run when nothing changes")
		return output.ChatRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestEditChat_PublicSetsUsername(t *testing.T) {
	name := "mychannel"
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:   "@grp",
		Username: &name,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.Username)
		require.Equal(t, "mychannel", *q.Username)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_PrivateSetsEmptyUsername(t *testing.T) {
	empty := ""
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:   "@grp",
		Username: &empty,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.Username, "private = pointer to empty string, not nil")
		require.Empty(t, *q.Username)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_RejectsShortUsername(t *testing.T) {
	short := "ab"
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:   "@grp",
		Username: &short,
	}, func(context.Context, actionchat.EditChatQuery) (output.ChatRow, error) {
		t.Fatal("edit must not run with a too-short username")
		return output.ChatRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestEditChat_RejectsEmptyTitle(t *testing.T) {
	empty := ""
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef: "@grp",
		Title:  &empty,
	}, func(context.Context, actionchat.EditChatQuery) (output.ChatRow, error) {
		t.Fatal("edit must not run with an empty title")
		return output.ChatRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}
