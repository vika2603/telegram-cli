package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func TestEditTopic_PassesOnlySetFields(t *testing.T) {
	_, err := actionchat.EditTopic(context.Background(), actionchat.EditTopicRequest{
		RawRef:  "@forum",
		TopicID: 5,
		Title:   strPtr("renamed"),
		Closed:  boolPtr(true),
	}, func(_ context.Context, q actionchat.EditTopicQuery) (output.TopicRow, error) {
		require.Equal(t, 5, q.TopicID)
		require.NotNil(t, q.Title)
		require.Equal(t, "renamed", *q.Title)
		require.NotNil(t, q.Closed)
		require.True(t, *q.Closed)
		require.Nil(t, q.Hidden, "hidden not passed -> stays nil")
		return output.TopicRow{ID: 5, Title: "renamed", Closed: true}, nil
	})
	require.NoError(t, err)
}

func TestEditTopic_RejectsNoChange(t *testing.T) {
	_, err := actionchat.EditTopic(context.Background(), actionchat.EditTopicRequest{
		RawRef: "@forum", TopicID: 5,
	}, func(context.Context, actionchat.EditTopicQuery) (output.TopicRow, error) {
		t.Fatal("edit must not run when nothing changes")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestEditTopic_RejectsBadTopicID(t *testing.T) {
	_, err := actionchat.EditTopic(context.Background(), actionchat.EditTopicRequest{
		RawRef: "@forum", TopicID: 0, Title: strPtr("x"),
	}, func(context.Context, actionchat.EditTopicQuery) (output.TopicRow, error) {
		t.Fatal("edit must not run with an invalid topic id")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestDeleteTopic_ConfirmsBeforeDispatch(t *testing.T) {
	called := false
	_, err := actionchat.DeleteTopic(context.Background(), actionchat.DeleteTopicRequest{
		RawRef:   "@forum",
		TopicID:  5,
		Prompter: &ui.StubPrompter{Answers: []any{true}},
	}, func(_ context.Context, q actionchat.DeleteTopicQuery) error {
		called = true
		require.Equal(t, 5, q.TopicID)
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestDeleteTopic_DeclineSkipsDispatch(t *testing.T) {
	_, err := actionchat.DeleteTopic(context.Background(), actionchat.DeleteTopicRequest{
		RawRef:   "@forum",
		TopicID:  5,
		Prompter: &ui.StubPrompter{Answers: []any{false}},
	}, func(context.Context, actionchat.DeleteTopicQuery) error {
		t.Fatal("delete must not run when the prompt is declined")
		return nil
	})
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestPinTopic_UnpinSetsPinnedFalse(t *testing.T) {
	_, err := actionchat.PinTopic(context.Background(), actionchat.PinTopicRequest{
		RawRef: "@forum", TopicID: 5, Unpin: true,
	}, func(_ context.Context, q actionchat.PinTopicQuery) (output.TopicRow, error) {
		require.False(t, q.Pinned)
		return output.TopicRow{ID: 5}, nil
	})
	require.NoError(t, err)
}
