package member_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/member"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNewList_FlagsAndArgs(t *testing.T) {
	var captured *member.Options
	f := runtime.NewTestInvocation(t)
	cmd := member.NewList(f, func(o *member.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"@ch", "--filter", "admins", "--limit", "100"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@ch", captured.RawRef)
	require.Equal(t, "admins", captured.Filter)
	require.Equal(t, 100, captured.Limit)
}

func TestRun_QWithRecentIsUsageError(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &member.Options{
		RawRef:    "@ch",
		Filter:    "recent",
		Q:         "alice",
		IOStreams: ios,
	}
	err := member.Run(context.Background(), opts)
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_Renders(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &member.Options{
		RawRef:    "@ch",
		Filter:    "recent",
		Limit:     30,
		IOStreams: ios,
		Fetch: func(context.Context, actionchat.MembersQuery) ([]output.MemberRow, error) {
			return []output.MemberRow{{UserID: 1, FirstName: "Alice", Role: "member"}}, nil
		},
	}
	require.NoError(t, member.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "Alice")
}
