package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestLinkDiscussion_ParsesBothRefs(t *testing.T) {
	row, err := actionchat.LinkDiscussion(context.Background(), actionchat.DiscussionRequest{
		RawChannel: "@chan",
		RawGroup:   "@grp",
	}, func(_ context.Context, q actionchat.DiscussionQuery) (output.DiscussionRow, error) {
		require.Equal(t, "chan", q.Channel.Value)
		require.Equal(t, "grp", q.Group.Value)
		require.False(t, q.Unlink)
		return output.DiscussionRow{Action: "link"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "link", row.Action)
}

func TestLinkDiscussion_RejectsBadGroupRef(t *testing.T) {
	_, err := actionchat.LinkDiscussion(context.Background(), actionchat.DiscussionRequest{
		RawChannel: "@chan",
		RawGroup:   "",
	}, func(context.Context, actionchat.DiscussionQuery) (output.DiscussionRow, error) {
		t.Fatal("must not dispatch with an invalid group ref")
		return output.DiscussionRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestUnlinkDiscussion_SetsUnlinkAndIgnoresGroup(t *testing.T) {
	row, err := actionchat.UnlinkDiscussion(context.Background(), actionchat.DiscussionRequest{
		RawChannel: "@chan",
	}, func(_ context.Context, q actionchat.DiscussionQuery) (output.DiscussionRow, error) {
		require.Equal(t, "chan", q.Channel.Value)
		require.True(t, q.Unlink)
		return output.DiscussionRow{Action: "unlink"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "unlink", row.Action)
}

func TestDiscussionCandidates_NilDoReturnsPrecondition(t *testing.T) {
	_, err := actionchat.DiscussionCandidates(context.Background(), nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}
