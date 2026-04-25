package members_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/members"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagsAndArgs(t *testing.T) {
	var captured *members.Options
	f := runtime.NewTestInvocation(t)
	cmd := members.New(f, func(o *members.Options) error {
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
	opts := &members.Options{
		RawRef:    "@ch",
		Filter:    "recent",
		Q:         "alice",
		IOStreams: ios,
	}
	err := members.Run(context.Background(), opts)
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_Renders(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &members.Options{
		RawRef:    "@ch",
		Filter:    "recent",
		Limit:     30,
		IOStreams: ios,
		Fetch: func(context.Context, actionchat.MembersQuery) ([]output.MemberRow, error) {
			return []output.MemberRow{{UserID: 1, FirstName: "Alice", Role: "member"}}, nil
		},
	}
	require.NoError(t, members.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "Alice")
}
