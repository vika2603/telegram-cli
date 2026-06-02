package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestApproveJoin_ParsesUsers(t *testing.T) {
	rows, err := actionchat.ApproveJoin(context.Background(), actionchat.JoinDecisionRequest{
		RawRef:   "@grp",
		RawUsers: []string{"@alice", "@bob"},
	}, func(_ context.Context, q actionchat.JoinDecisionQuery) ([]output.JoinResultRow, error) {
		require.True(t, q.Approved)
		require.False(t, q.All)
		require.Len(t, q.Users, 2)
		require.Equal(t, "alice", q.Users[0].Value)
		return []output.JoinResultRow{{Action: "approve"}, {Action: "approve"}}, nil
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

func TestDenyJoin_AllSetsFlag(t *testing.T) {
	_, err := actionchat.DenyJoin(context.Background(), actionchat.JoinDecisionRequest{
		RawRef: "@grp",
		All:    true,
	}, func(_ context.Context, q actionchat.JoinDecisionQuery) ([]output.JoinResultRow, error) {
		require.False(t, q.Approved)
		require.True(t, q.All)
		require.Empty(t, q.Users)
		return []output.JoinResultRow{{Action: "deny", All: true}}, nil
	})
	require.NoError(t, err)
}

func TestApproveJoin_AllAndUsersMutuallyExclusive(t *testing.T) {
	_, err := actionchat.ApproveJoin(context.Background(), actionchat.JoinDecisionRequest{
		RawRef: "@grp", RawUsers: []string{"@alice"}, All: true,
	}, func(context.Context, actionchat.JoinDecisionQuery) ([]output.JoinResultRow, error) {
		t.Fatal("must not dispatch")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestApproveJoin_RequiresUserOrAll(t *testing.T) {
	_, err := actionchat.ApproveJoin(context.Background(), actionchat.JoinDecisionRequest{
		RawRef: "@grp",
	}, func(context.Context, actionchat.JoinDecisionQuery) ([]output.JoinResultRow, error) {
		t.Fatal("must not dispatch")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestJoinList_ParsesRef(t *testing.T) {
	_, err := actionchat.JoinList(context.Background(), actionchat.JoinListRequest{
		RawRef: "@grp", Limit: 50,
	}, func(_ context.Context, q actionchat.JoinListQuery) ([]output.MemberRow, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.Equal(t, 50, q.Limit)
		return nil, nil
	})
	require.NoError(t, err)
}
