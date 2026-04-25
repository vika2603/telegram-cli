package react_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/msg/react"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *react.Options
	f := runtime.NewTestInvocation(t)
	cmd := react.New(f, func(o *react.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@a:10", "--emoji", "👍"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@a:10", captured.RawMessageRef)
	require.Equal(t, "👍", captured.Emoji)
}

func TestRun_NeitherFlagRejected(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &react.Options{RawMessageRef: "@a:1", IOStreams: ios}
	err := react.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_BothFlagsRejected(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &react.Options{RawMessageRef: "@a:1", Emoji: "👍", Clear: true, IOStreams: ios}
	err := react.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_NilReactClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &react.Options{RawMessageRef: "@a:1", Emoji: "👍", IOStreams: ios}
	err := react.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedReact(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &react.Options{
		RawMessageRef: "@a:10", Emoji: "👍", IOStreams: ios,
		React: func(_ context.Context, a actionmessage.ReactQuery) (output.SendResultRow, error) {
			require.Equal(t, "👍", a.Emoji)
			require.Equal(t, 10, a.MessageID)
			require.False(t, a.Clear)
			return output.SendResultRow{Action: "react", MessageID: 10, ChatID: 42}, nil
		},
	}
	require.NoError(t, react.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "react")
	require.Contains(t, stdout.String(), "10")
}

func TestRun_StubbedClear(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &react.Options{
		RawMessageRef: "@a:10", Clear: true, IOStreams: ios,
		React: func(_ context.Context, a actionmessage.ReactQuery) (output.SendResultRow, error) {
			require.True(t, a.Clear)
			require.Empty(t, a.Emoji)
			return output.SendResultRow{Action: "react", MessageID: 10, ChatID: 42}, nil
		},
	}
	require.NoError(t, react.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "react")
}
