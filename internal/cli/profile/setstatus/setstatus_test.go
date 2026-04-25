package setstatus_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/profile/setstatus"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *setstatus.Options
	f := runtime.NewTestInvocation(t)
	cmd := setstatus.New(f, func(o *setstatus.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"online"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "online", captured.State)
}

func TestRun_NilUpdateClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &setstatus.Options{State: "online", IOStreams: ios}
	err := setstatus.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_RejectBadState(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &setstatus.Options{
		State: "bogus", IOStreams: ios,
		Update: func(_ context.Context, _ bool) (output.ProfileRow, error) {
			t.Fatal("Update must not run for invalid state")
			return output.ProfileRow{}, nil
		},
	}
	err := setstatus.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_StubbedUpdateOnline(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &setstatus.Options{
		State: "online", IOStreams: ios,
		Update: func(_ context.Context, offline bool) (output.ProfileRow, error) {
			require.False(t, offline)
			return output.ProfileRow{Action: "set-status", Status: "online"}, nil
		},
	}
	require.NoError(t, setstatus.Run(context.Background(), opts))
	s := stdout.String()
	require.Contains(t, s, `"action":"set-status"`)
	require.Contains(t, s, `"status":"online"`)
}

func TestRun_StubbedUpdateOffline(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &setstatus.Options{
		State: "offline", IOStreams: ios,
		Update: func(_ context.Context, offline bool) (output.ProfileRow, error) {
			require.True(t, offline)
			return output.ProfileRow{Action: "set-status", Status: "offline"}, nil
		},
	}
	require.NoError(t, setstatus.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), `"status":"offline"`)
}
