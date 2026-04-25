package read_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/read"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *read.Options
	f := runtime.NewTestInvocation(t)
	cmd := read.New(f, func(o *read.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@a", "--max-id", "100"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@a", captured.RawRef)
	require.Equal(t, 100, captured.MaxID)
}

func TestRun_NilReadClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &read.Options{RawRef: "@a", IOStreams: ios}
	err := read.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedRead(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	called := false
	opts := &read.Options{
		RawRef: "@a", MaxID: 77, IOStreams: ios,
		Read: func(_ context.Context, a actionchat.ReadQuery) error {
			called = true
			require.Equal(t, "a", a.Ref.Value)
			require.Equal(t, 77, a.MaxID)
			return nil
		},
	}
	require.NoError(t, read.Run(context.Background(), opts))
	require.True(t, called)
	require.Equal(t, "read\t@a\n", stdout.String())
}

func TestRun_MaxIDDefaultZero(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &read.Options{
		RawRef: "@a", IOStreams: ios,
		Read: func(_ context.Context, a actionchat.ReadQuery) error {
			require.Equal(t, 0, a.MaxID)
			return nil
		},
	}
	require.NoError(t, read.Run(context.Background(), opts))
}
