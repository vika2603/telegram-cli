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

func TestInviteLinkCreate_ParsesExpireDuration(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	row, err := actionchat.InviteLinkCreate(context.Background(), actionchat.InviteLinkCreateRequest{
		RawRef:     "@grp",
		Expire:     "24h",
		UsageLimit: 5,
		Now:        now,
	}, func(_ context.Context, q actionchat.InviteLinkCreateQuery) (output.InviteLinkRow, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.Equal(t, int(now.Add(24*time.Hour).Unix()), q.ExpireDate)
		require.Equal(t, 5, q.UsageLimit)
		return output.InviteLinkRow{Action: "create", Link: "https://t.me/+abc"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "create", row.Action)
}

func TestInviteLinkCreate_RejectsNegativeUsageLimit(t *testing.T) {
	_, err := actionchat.InviteLinkCreate(context.Background(), actionchat.InviteLinkCreateRequest{
		RawRef: "@grp", UsageLimit: -1,
	}, func(context.Context, actionchat.InviteLinkCreateQuery) (output.InviteLinkRow, error) {
		t.Fatal("must not dispatch")
		return output.InviteLinkRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestInviteLinkList_ParsesAdmin(t *testing.T) {
	_, err := actionchat.InviteLinkList(context.Background(), actionchat.InviteLinkListRequest{
		RawRef: "@grp", RawAdmin: "@alice", Revoked: true,
	}, func(_ context.Context, q actionchat.InviteLinkListQuery) ([]output.InviteLinkRow, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.Equal(t, "alice", q.Admin.Value)
		require.True(t, q.Revoked)
		return nil, nil
	})
	require.NoError(t, err)
}

func TestInviteLinkRevoke_RequiresLink(t *testing.T) {
	_, err := actionchat.InviteLinkRevoke(context.Background(), actionchat.InviteLinkRequest{
		RawRef: "@grp",
	}, func(context.Context, actionchat.InviteLinkQuery) (output.InviteLinkRow, error) {
		t.Fatal("must not dispatch without a link")
		return output.InviteLinkRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}
