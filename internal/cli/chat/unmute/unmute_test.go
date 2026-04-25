package unmute_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/unmute"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	var captured *unmute.Options
	cmd := unmute.New(f, func(o *unmute.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@durov"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, "@durov", captured.RawRef)
}

func TestRun_NilDoReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &unmute.Options{RawRef: "@durov", IOStreams: ios}
	err := unmute.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_InvalidRefIsUsage(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &unmute.Options{
		RawRef: "", IOStreams: ios,
		Do: func(context.Context, actionchat.UnmuteQuery) (output.ChatMuteRow, error) {
			t.Fatal("Do must not run when ref invalid")
			return output.ChatMuteRow{}, nil
		},
	}
	err := unmute.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_StubbedOK(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &unmute.Options{
		RawRef: "@durov", IOStreams: ios,
		Do: func(_ context.Context, args actionchat.UnmuteQuery) (output.ChatMuteRow, error) {
			require.Equal(t, "durov", args.Ref.Value)
			return output.ChatMuteRow{Action: "unmute"}, nil
		},
	}
	require.NoError(t, unmute.Run(context.Background(), opts))
}
