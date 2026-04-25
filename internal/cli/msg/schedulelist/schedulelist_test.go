package schedulelist_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/msg/schedulelist"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *schedulelist.Options
	f := runtime.NewTestInvocation(t)
	cmd := schedulelist.New(f, func(o *schedulelist.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@a"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@a", captured.RawRef)
}

func TestRun_NilFetchClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &schedulelist.Options{RawRef: "@a", IOStreams: ios}
	err := schedulelist.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedFetch(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &schedulelist.Options{
		RawRef: "@a", IOStreams: ios,
		Fetch: func(_ context.Context, a actionmessage.ScheduledListQuery) ([]output.ScheduledMessageRow, error) {
			require.Equal(t, "a", a.Ref.Value)
			return []output.ScheduledMessageRow{
				{ID: 8001, Date: "2026-05-01T00:00:00Z", ScheduledFor: "2026-05-02T09:00:00Z", Text: "hi"},
			}, nil
		},
	}
	require.NoError(t, schedulelist.Run(context.Background(), opts))
	s := stdout.String()
	require.Contains(t, s, "8001")
	require.Contains(t, s, "hi")
}

func TestRun_EmptyRowsRenders(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &schedulelist.Options{
		RawRef: "@a", IOStreams: ios,
		Fetch: func(_ context.Context, _ actionmessage.ScheduledListQuery) ([]output.ScheduledMessageRow, error) {
			return nil, nil
		},
	}
	require.NoError(t, schedulelist.Run(context.Background(), opts))
}
