package pin_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/msg/pin"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *pin.Options
	f := runtime.NewTestInvocation(t)
	cmd := pin.New(f, func(o *pin.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@a:42", "--silent"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@a:42", captured.RawMessageRef)
	require.True(t, captured.Silent)
	require.False(t, captured.Unpin)
}

// TestNew_JSONFlag verifies that the pin command now accepts --json
// (previously rejected with "unknown flag: --json"). The actual JSON
// shape is exercised through output/json_flags_test.go; here we only
// confirm the flag is wired up.
func TestNew_JSONFlag(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := pin.New(f, func(*pin.Options) error { return nil })
	require.NotNil(t, cmd.Flag("json"), "--json must be registered on tg msg pin")
	cmdUnpin := pin.NewUnpin(f, func(*pin.Options) error { return nil })
	require.NotNil(t, cmdUnpin.Flag("json"), "--json must be registered on tg msg unpin")
}

func TestRun_NilDoClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &pin.Options{RawMessageRef: "@a:1", IOStreams: ios}
	err := pin.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedPin(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	called := false
	opts := &pin.Options{
		RawMessageRef: "@a:7", IOStreams: ios,
		Do: func(_ context.Context, a actionmessage.PinQuery) error {
			called = true
			require.Equal(t, 7, a.MessageID)
			require.False(t, a.Unpin)
			return nil
		},
	}
	require.NoError(t, pin.Run(context.Background(), opts))
	require.True(t, called)
	require.Equal(t, "ACTION\tMESSAGE_ID\tCHAT_ID\tDATE\npin\t7\t\t\n", stdout.String())
}

func TestRun_StubbedUnpin(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &pin.Options{
		RawMessageRef: "@a:9", Unpin: true, IOStreams: ios,
		Do: func(_ context.Context, a actionmessage.PinQuery) error {
			require.True(t, a.Unpin)
			return nil
		},
	}
	require.NoError(t, pin.Run(context.Background(), opts))
	require.Equal(t, "ACTION\tMESSAGE_ID\tCHAT_ID\tDATE\nunpin\t9\t\t\n", stdout.String())
}
