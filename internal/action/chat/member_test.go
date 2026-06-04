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

func TestInvite_ParsesGroupAndUsers(t *testing.T) {
	rows, err := actionchat.Invite(context.Background(), actionchat.InviteRequest{
		RawRef:   "@grp",
		RawUsers: []string{"@alice", "@bob"},
	}, func(_ context.Context, q actionchat.InviteQuery) ([]output.InviteRow, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.Len(t, q.Users, 2)
		require.Equal(t, "alice", q.Users[0].Value)
		require.Equal(t, "bob", q.Users[1].Value)
		return []output.InviteRow{
			{Peer: output.PeerRef{Ref: "@alice"}, Invited: true},
			{Peer: output.PeerRef{Ref: "@bob"}, Invited: false, SkipReason: "privacy_restricted"},
		}, nil
	})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.True(t, rows[0].Invited)
	require.False(t, rows[1].Invited)
	require.Equal(t, "privacy_restricted", rows[1].SkipReason)
}

func TestInvite_RequiresAtLeastOneUser(t *testing.T) {
	_, err := actionchat.Invite(context.Background(), actionchat.InviteRequest{
		RawRef: "@grp",
	}, func(context.Context, actionchat.InviteQuery) ([]output.InviteRow, error) {
		t.Fatal("invite must not run without users")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestBan_ConfirmsBeforeDispatch(t *testing.T) {
	called := false
	_, err := actionchat.Ban(context.Background(), actionchat.BanRequest{
		RawRef:   "@grp",
		RawUser:  "@alice",
		Prompter: &ui.StubPrompter{Answers: []any{true}},
	}, func(_ context.Context, q actionchat.BanQuery) (output.PeerRef, error) {
		called = true
		require.Equal(t, "alice", q.User.Value)
		require.False(t, q.Unban)
		return output.PeerRef{}, nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestBan_DeclineSkipsDispatch(t *testing.T) {
	_, err := actionchat.Ban(context.Background(), actionchat.BanRequest{
		RawRef:   "@grp",
		RawUser:  "@alice",
		Prompter: &ui.StubPrompter{Answers: []any{false}},
	}, func(context.Context, actionchat.BanQuery) (output.PeerRef, error) {
		t.Fatal("ban must not run when the prompt is declined")
		return output.PeerRef{}, nil
	})
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestUnban_SkipsConfirmation(t *testing.T) {
	called := false
	_, err := actionchat.Ban(context.Background(), actionchat.BanRequest{
		RawRef:  "@grp",
		RawUser: "@alice",
		Unban:   true,
		// no Prompter: unban must not prompt
	}, func(_ context.Context, q actionchat.BanQuery) (output.PeerRef, error) {
		called = true
		require.True(t, q.Unban)
		return output.PeerRef{}, nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestPromote_TitlePassesThrough(t *testing.T) {
	_, err := actionchat.Promote(context.Background(), actionchat.PromoteRequest{
		RawRef:   "@grp",
		RawUser:  "@alice",
		Title:    "客服",
		SetTitle: true,
	}, func(_ context.Context, q actionchat.PromoteQuery) (output.RightsRow, error) {
		require.Equal(t, "客服", q.Title)
		require.True(t, q.SetTitle)
		require.False(t, q.Demote)
		return output.RightsRow{}, nil
	})
	require.NoError(t, err)
}

func TestPromote_RejectsLongTitle(t *testing.T) {
	_, err := actionchat.Promote(context.Background(), actionchat.PromoteRequest{
		RawRef:  "@grp",
		RawUser: "@alice",
		Title:   "this-title-is-way-too-long-for-telegram",
	}, func(context.Context, actionchat.PromoteQuery) (output.RightsRow, error) {
		t.Fatal("must not dispatch with an over-long title")
		return output.RightsRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestPromote_DemoteFlagPassesThrough(t *testing.T) {
	_, err := actionchat.Promote(context.Background(), actionchat.PromoteRequest{
		RawRef:  "@grp",
		RawUser: "@alice",
		Demote:  true,
	}, func(_ context.Context, q actionchat.PromoteQuery) (output.RightsRow, error) {
		require.Equal(t, "alice", q.User.Value)
		require.True(t, q.Demote)
		return output.RightsRow{}, nil
	})
	require.NoError(t, err)
}

func TestPromote_RightsPassThrough(t *testing.T) {
	_, err := actionchat.Promote(context.Background(), actionchat.PromoteRequest{
		RawRef:  "@grp",
		RawUser: "@alice",
		Rights:  []string{"pin", "delete"},
	}, func(_ context.Context, q actionchat.PromoteQuery) (output.RightsRow, error) {
		require.Equal(t, []string{"pin", "delete"}, q.Rights)
		require.False(t, q.Demote)
		return output.RightsRow{}, nil
	})
	require.NoError(t, err)
}

func TestPromote_RejectsUnknownRight(t *testing.T) {
	_, err := actionchat.Promote(context.Background(), actionchat.PromoteRequest{
		RawRef:  "@grp",
		RawUser: "@alice",
		Rights:  []string{"pin", "bogus"},
	}, func(context.Context, actionchat.PromoteQuery) (output.RightsRow, error) {
		t.Fatal("must not dispatch with an unknown right")
		return output.RightsRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestPromote_RejectsRightsWithDemote(t *testing.T) {
	_, err := actionchat.Promote(context.Background(), actionchat.PromoteRequest{
		RawRef:  "@grp",
		RawUser: "@alice",
		Demote:  true,
		Rights:  []string{"pin"},
	}, func(context.Context, actionchat.PromoteQuery) (output.RightsRow, error) {
		t.Fatal("must not dispatch --rights with demote")
		return output.RightsRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}
