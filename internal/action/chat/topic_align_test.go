package chat_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// InfoTopic tests

func TestInfoTopic_RejectsBadTopicID(t *testing.T) {
	_, err := actionchat.InfoTopic(context.Background(), actionchat.TopicInfoRequest{
		RawRef:  "@forum",
		TopicID: 0,
	}, func(context.Context, actionchat.TopicInfoQuery) (output.TopicRow, error) {
		t.Fatal("info must not run with an invalid topic id")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestInfoTopic_RejectsBadRef(t *testing.T) {
	_, err := actionchat.InfoTopic(context.Background(), actionchat.TopicInfoRequest{
		RawRef:  "",
		TopicID: 5,
	}, func(context.Context, actionchat.TopicInfoQuery) (output.TopicRow, error) {
		t.Fatal("info must not run with an invalid ref")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestInfoTopic_NilDoReturnsPrecondition(t *testing.T) {
	_, err := actionchat.InfoTopic(context.Background(), actionchat.TopicInfoRequest{
		RawRef:  "@forum",
		TopicID: 5,
	}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestInfoTopic_PassesFieldsThrough(t *testing.T) {
	row, err := actionchat.InfoTopic(context.Background(), actionchat.TopicInfoRequest{
		RawRef:  "@forum",
		TopicID: 7,
	}, func(_ context.Context, q actionchat.TopicInfoQuery) (output.TopicRow, error) {
		require.Equal(t, 7, q.TopicID)
		return output.TopicRow{ID: 7, Title: "General"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 7, row.ID)
	require.Equal(t, "General", row.Title)
}

// MuteTopic tests

func TestMuteTopic_RejectsBadTopicID(t *testing.T) {
	_, err := actionchat.MuteTopic(context.Background(), actionchat.MuteTopicRequest{
		RawRef:  "@forum",
		TopicID: 0,
	}, func(context.Context, actionchat.MuteTopicQuery) (output.TopicRow, error) {
		t.Fatal("mute must not run with an invalid topic id")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestMuteTopic_RejectsBadRef(t *testing.T) {
	_, err := actionchat.MuteTopic(context.Background(), actionchat.MuteTopicRequest{
		RawRef:  "",
		TopicID: 5,
	}, func(context.Context, actionchat.MuteTopicQuery) (output.TopicRow, error) {
		t.Fatal("mute must not run with an invalid ref")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestMuteTopic_NilDoReturnsPrecondition(t *testing.T) {
	_, err := actionchat.MuteTopic(context.Background(), actionchat.MuteTopicRequest{
		RawRef:  "@forum",
		TopicID: 5,
	}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestMuteTopic_DefaultsToForever(t *testing.T) {
	const forever = 1<<31 - 1 // math.MaxInt32
	row, err := actionchat.MuteTopic(context.Background(), actionchat.MuteTopicRequest{
		RawRef:  "@forum",
		TopicID: 5,
	}, func(_ context.Context, q actionchat.MuteTopicQuery) (output.TopicRow, error) {
		require.Equal(t, 5, q.TopicID)
		require.Equal(t, forever, q.MuteUntil)
		return output.TopicRow{ID: 5}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 5, row.ID)
}

func TestMuteTopic_DurationComputesFutureTimestamp(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := actionchat.MuteTopic(context.Background(), actionchat.MuteTopicRequest{
		RawRef:   "@forum",
		TopicID:  5,
		Duration: "1h",
		Now:      now,
	}, func(_ context.Context, q actionchat.MuteTopicQuery) (output.TopicRow, error) {
		require.Equal(t, int(now.Add(time.Hour).Unix()), q.MuteUntil)
		return output.TopicRow{ID: 5}, nil
	})
	require.NoError(t, err)
}

func TestMuteTopic_UnmutePassesZero(t *testing.T) {
	_, err := actionchat.MuteTopic(context.Background(), actionchat.MuteTopicRequest{
		RawRef:  "@forum",
		TopicID: 5,
		Unmute:  true,
	}, func(_ context.Context, q actionchat.MuteTopicQuery) (output.TopicRow, error) {
		require.Equal(t, 0, q.MuteUntil)
		return output.TopicRow{ID: 5}, nil
	})
	require.NoError(t, err)
}

// ReadTopic tests

func TestReadTopic_RejectsBadTopicID(t *testing.T) {
	_, err := actionchat.ReadTopic(context.Background(), actionchat.ReadTopicRequest{
		RawRef:  "@forum",
		TopicID: 0,
	}, func(context.Context, actionchat.ReadTopicQuery) (output.TopicRow, error) {
		t.Fatal("read must not run with an invalid topic id")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestReadTopic_RejectsBadRef(t *testing.T) {
	_, err := actionchat.ReadTopic(context.Background(), actionchat.ReadTopicRequest{
		RawRef:  "",
		TopicID: 5,
	}, func(context.Context, actionchat.ReadTopicQuery) (output.TopicRow, error) {
		t.Fatal("read must not run with an invalid ref")
		return output.TopicRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestReadTopic_NilDoReturnsPrecondition(t *testing.T) {
	_, err := actionchat.ReadTopic(context.Background(), actionchat.ReadTopicRequest{
		RawRef:  "@forum",
		TopicID: 5,
	}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestReadTopic_PassesFieldsThrough(t *testing.T) {
	row, err := actionchat.ReadTopic(context.Background(), actionchat.ReadTopicRequest{
		RawRef:  "@forum",
		TopicID: 9,
	}, func(_ context.Context, q actionchat.ReadTopicQuery) (output.TopicRow, error) {
		require.Equal(t, 9, q.TopicID)
		return output.TopicRow{ID: 9}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 9, row.ID)
}
