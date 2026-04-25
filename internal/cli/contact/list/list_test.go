package list_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/cli/contact/list"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *list.Options
	f := runtime.NewTestInvocation(t)
	cmd := list.New(f, func(o *list.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"--blocked", "--mutual-only", "--bots"})
	require.NoError(t, cmd.Execute())
	require.True(t, captured.Blocked)
	require.True(t, captured.MutualOnly)
	require.True(t, captured.Bots)
}

func TestRun_NilFetchClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &list.Options{IOStreams: ios}
	err := list.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedFetch(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &list.Options{
		IOStreams: ios,
		Fetch: func(_ context.Context, a actioncontact.ListQuery) ([]output.ContactRow, error) {
			require.False(t, a.Blocked)
			return []output.ContactRow{
				{ID: 1, FirstName: "Alice", Username: "alice", Phone: "+1", Mutual: true},
			}, nil
		},
	}
	require.NoError(t, list.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "Alice")
	require.Contains(t, stdout.String(), "alice")
}

func TestRun_StubbedFetchBlocked(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &list.Options{
		Blocked: true, IOStreams: ios,
		Fetch: func(_ context.Context, a actioncontact.ListQuery) ([]output.ContactRow, error) {
			require.True(t, a.Blocked)
			return []output.ContactRow{{ID: 9, FirstName: "Bob", Blocked: true}}, nil
		},
	}
	require.NoError(t, list.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "Bob")
}
