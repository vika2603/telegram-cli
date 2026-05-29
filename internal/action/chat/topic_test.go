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

func TestTopics_NormalizesRefAndDefaultsLimit(t *testing.T) {
	rows, err := actionchat.Topics(context.Background(), actionchat.TopicsRequest{
		RawRef: "@forum",
	}, func(_ context.Context, q actionchat.TopicsQuery) ([]output.TopicRow, error) {
		require.Equal(t, "forum", q.Ref.Value)
		require.Equal(t, 100, q.Limit)
		return []output.TopicRow{{ID: 1, Title: "General"}}, nil
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestCreateTopic_PassesFieldsThrough(t *testing.T) {
	_, err := actionchat.CreateTopic(context.Background(), actionchat.CreateTopicRequest{
		RawRef:    "@forum",
		Title:     "Bugs",
		IconColor: 0x6FB9F0,
		RandomID:  555,
	}, func(_ context.Context, q actionchat.CreateTopicQuery) (output.TopicRow, error) {
		require.Equal(t, "forum", q.Ref.Value)
		require.Equal(t, "Bugs", q.Title)
		require.Equal(t, 0x6FB9F0, q.IconColor)
		require.Equal(t, int64(555), q.RandomID)
		return output.TopicRow{ID: 10, Title: "Bugs"}, nil
	})
	require.NoError(t, err)
}

func TestCreateTopic_RejectsEmptyTitle(t *testing.T) {
	_, err := actionchat.CreateTopic(context.Background(), actionchat.CreateTopicRequest{
		RawRef: "@forum",
		Title:  "",
	}, func(context.Context, actionchat.CreateTopicQuery) (output.TopicRow, error) {
		t.Fatal("create must not run with an empty title")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestCreateTopic_RejectsOverlongTitle(t *testing.T) {
	_, err := actionchat.CreateTopic(context.Background(), actionchat.CreateTopicRequest{
		RawRef: "@forum",
		Title:  strings.Repeat("x", 129),
	}, func(context.Context, actionchat.CreateTopicQuery) (output.TopicRow, error) {
		t.Fatal("create must not run with an overlong title")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}
