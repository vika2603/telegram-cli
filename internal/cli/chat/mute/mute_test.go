package mute_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/mute"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing_Forever(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	var captured *mute.Options
	cmd := mute.New(f, func(o *mute.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@durov", "--forever"})
	require.NoError(t, cmd.Execute())
	require.True(t, captured.Forever)
}

func TestNew_FlagParsing_Duration(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	var captured *mute.Options
	cmd := mute.New(f, func(o *mute.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@durov", "--duration", "3d"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "3d", captured.Duration)
}

func TestRun_MutexDurationAndUntil(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &mute.Options{
		RawRef: "@durov", IOStreams: ios,
		Duration: "1h", Until: "2026-05-01T00:00:00Z",
		Do: func(context.Context, actionchat.MuteQuery) (output.ChatMuteRow, error) {
			t.Fatal("Do must not run when flags conflict")
			return output.ChatMuteRow{}, nil
		},
	}
	err := mute.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_NilDoReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &mute.Options{RawRef: "@durov", IOStreams: ios, Forever: true}
	err := mute.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_ForeverPassesMaxInt32(t *testing.T) {
	ios, _, _, _ := ui.Test()
	var seen actionchat.MuteQuery
	opts := &mute.Options{
		RawRef: "@durov", IOStreams: ios, Forever: true,
		Do: func(_ context.Context, a actionchat.MuteQuery) (output.ChatMuteRow, error) {
			seen = a
			return output.ChatMuteRow{Action: "mute"}, nil
		},
	}
	require.NoError(t, mute.Run(context.Background(), opts))
	require.Equal(t, int32(2147483647), seen.MuteUntil) // math.MaxInt32
}

func TestRun_DurationBecomesFutureTimestamp(t *testing.T) {
	ios, _, _, _ := ui.Test()
	var seen actionchat.MuteQuery
	opts := &mute.Options{
		RawRef: "@durov", IOStreams: ios, Duration: "1h",
		Do: func(_ context.Context, a actionchat.MuteQuery) (output.ChatMuteRow, error) {
			seen = a
			return output.ChatMuteRow{Action: "mute"}, nil
		},
	}
	require.NoError(t, mute.Run(context.Background(), opts))
	// Should be roughly 1h in the future. Allow a window.
	now := int32(time.Now().Unix())
	require.InDelta(t, float64(now+3600), float64(seen.MuteUntil), 10)
}

func TestRun_UntilInPastIsUsage(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &mute.Options{
		RawRef: "@durov", IOStreams: ios,
		Until: "2000-01-01T00:00:00Z",
		Do: func(context.Context, actionchat.MuteQuery) (output.ChatMuteRow, error) {
			t.Fatal("Do must not run when --until is in the past")
			return output.ChatMuteRow{}, nil
		},
	}
	err := mute.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestParseDuration_DaysSuffix(t *testing.T) {
	d, err := actionchat.ParseDurationWithDays("3d")
	require.NoError(t, err)
	require.Equal(t, 72*time.Hour, d)
	d, err = actionchat.ParseDurationWithDays("90m")
	require.NoError(t, err)
	require.Equal(t, 90*time.Minute, d)
	_, err = actionchat.ParseDurationWithDays("bogus")
	require.Error(t, err)
}
