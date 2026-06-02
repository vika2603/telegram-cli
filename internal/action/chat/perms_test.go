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

func TestRestrict_ParsesKeysAndUntil(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	row, err := actionchat.Restrict(context.Background(), actionchat.RestrictRequest{
		RawRef:  "@grp",
		RawUser: "@alice",
		Deny:    []string{"send", "media"},
		Until:   "1h",
		Now:     now,
	}, func(_ context.Context, q actionchat.RestrictQuery) (output.RightsRow, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.Equal(t, "alice", q.User.Value)
		require.Equal(t, []string{"send", "media"}, q.Deny)
		require.Equal(t, int(now.Add(time.Hour).Unix()), q.UntilDate)
		require.False(t, q.Unrestrict)
		return output.RightsRow{Action: "set-perms"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "set-perms", row.Action)
}

func TestRestrict_RejectsUnknownKey(t *testing.T) {
	_, err := actionchat.Restrict(context.Background(), actionchat.RestrictRequest{
		RawRef: "@grp", RawUser: "@alice", Deny: []string{"bogus"},
	}, func(context.Context, actionchat.RestrictQuery) (output.RightsRow, error) {
		t.Fatal("must not dispatch")
		return output.RightsRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRestrict_RequiresAllowOrDeny(t *testing.T) {
	_, err := actionchat.Restrict(context.Background(), actionchat.RestrictRequest{
		RawRef: "@grp", RawUser: "@alice",
	}, func(context.Context, actionchat.RestrictQuery) (output.RightsRow, error) {
		t.Fatal("must not dispatch")
		return output.RightsRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestUnrestrict_SetsFlag(t *testing.T) {
	_, err := actionchat.Unrestrict(context.Background(), actionchat.RestrictRequest{
		RawRef: "@grp", RawUser: "@alice",
	}, func(_ context.Context, q actionchat.RestrictQuery) (output.RightsRow, error) {
		require.True(t, q.Unrestrict)
		return output.RightsRow{Action: "unset-perms"}, nil
	})
	require.NoError(t, err)
}

func TestPerms_ValidatesAndRequiresKeys(t *testing.T) {
	_, err := actionchat.Perms(context.Background(), actionchat.PermsRequest{
		RawRef: "@grp", Deny: []string{"polls", "links"},
	}, func(_ context.Context, q actionchat.PermsQuery) (output.RightsRow, error) {
		require.Equal(t, []string{"polls", "links"}, q.Deny)
		return output.RightsRow{Action: "perms"}, nil
	})
	require.NoError(t, err)

	_, err = actionchat.Perms(context.Background(), actionchat.PermsRequest{RawRef: "@grp"},
		func(context.Context, actionchat.PermsQuery) (output.RightsRow, error) {
			t.Fatal("must not dispatch")
			return output.RightsRow{}, nil
		})
	require.ErrorIs(t, err, command.ErrUsage)
}
