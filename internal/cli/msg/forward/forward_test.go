package forward_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/msg/forward"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *forward.Options
	f := runtime.NewTestInvocation(t)
	cmd := forward.New(f, func(o *forward.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@src:100", "@src:101", "--to", "@dst", "--silent"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, []string{"@src:100", "@src:101"}, captured.RawMessageRefs)
	require.Equal(t, "@dst", captured.RawTo)
	require.True(t, captured.Silent)
}

func TestRun_RequiresTo(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &forward.Options{RawMessageRefs: []string{"@a:1"}, IOStreams: ios}
	err := forward.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_NilForwardClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &forward.Options{RawMessageRefs: []string{"@a:1"}, RawTo: "@b", IOStreams: ios}
	err := forward.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedForward(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &forward.Options{
		RawMessageRefs: []string{"@a:5", "@a:6"}, RawTo: "@b",
		IOStreams: ios,
		Forward: func(_ context.Context, a actionmessage.ForwardQuery) (output.SendResultRow, error) {
			require.Equal(t, "a", a.From.Value)
			require.Equal(t, "b", a.To.Value)
			require.Equal(t, []int{5, 6}, a.IDs)
			return output.SendResultRow{Action: "forward", MessageID: 5, ChatID: 20}, nil
		},
	}
	require.NoError(t, forward.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "forward")
}
