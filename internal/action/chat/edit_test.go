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

// --- New toggle field tests ---

func TestEditChat_ForumPassedThrough(t *testing.T) {
	v := true
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef: "@grp",
		Forum:  &v,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.Forum)
		require.True(t, *q.Forum)
		require.Nil(t, q.Title)
		require.Nil(t, q.About)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_ForumFalsePassedThrough(t *testing.T) {
	v := false
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef: "@grp",
		Forum:  &v,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.Forum)
		require.False(t, *q.Forum)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_SlowModePassedThrough(t *testing.T) {
	secs := 30
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:   "@grp",
		SlowMode: &secs,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.SlowMode)
		require.Equal(t, 30, *q.SlowMode)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_SlowModeZeroDisables(t *testing.T) {
	zero := 0
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:   "@grp",
		SlowMode: &zero,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.SlowMode)
		require.Equal(t, 0, *q.SlowMode)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_RejectsNegativeSlowMode(t *testing.T) {
	neg := -1
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:   "@grp",
		SlowMode: &neg,
	}, func(context.Context, actionchat.EditChatQuery) (output.ChatRow, error) {
		t.Fatal("edit must not run with negative slow mode")
		return output.ChatRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestEditChat_HideMembersPassedThrough(t *testing.T) {
	v := true
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:      "@grp",
		HideMembers: &v,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.HideMembers)
		require.True(t, *q.HideMembers)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_HideHistoryPassedThrough(t *testing.T) {
	v := false
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:      "@grp",
		HideHistory: &v,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.HideHistory)
		require.False(t, *q.HideHistory)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_NoForwardsPassedThrough(t *testing.T) {
	v := true
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:     "@grp",
		NoForwards: &v,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.NoForwards)
		require.True(t, *q.NoForwards)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_SignaturesPassedThrough(t *testing.T) {
	v := true
	_, err := actionchat.EditChat(context.Background(), actionchat.EditChatRequest{
		RawRef:     "@grp",
		Signatures: &v,
	}, func(_ context.Context, q actionchat.EditChatQuery) (output.ChatRow, error) {
		require.NotNil(t, q.Signatures)
		require.True(t, *q.Signatures)
		return output.ChatRow{}, nil
	})
	require.NoError(t, err)
}

func TestEditChat_NothingToChangeWithAllNil(t *testing.T) {
	// Explicitly verify the guard fires when every new field is also nil.
	_, err := actionchat.NormalizeEditChat(actionchat.EditChatRequest{RawRef: "@grp"})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestEditChat_OnlyToggleCountsAsSomethingToChange(t *testing.T) {
	v := true
	// Forum alone (no title/about/username) must not trigger the "nothing to change" guard.
	_, err := actionchat.NormalizeEditChat(actionchat.EditChatRequest{
		RawRef: "@grp",
		Forum:  &v,
	})
	require.NoError(t, err)
}
