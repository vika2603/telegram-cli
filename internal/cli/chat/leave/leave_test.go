package leave_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/leave"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	var captured *leave.Options
	cmd := leave.New(f, func(o *leave.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@durov", "--yes"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, "@durov", captured.RawRef)
	require.True(t, captured.Yes)
}

func TestRun_NilDoReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &leave.Options{RawRef: "@durov", IOStreams: ios, Yes: true}
	err := leave.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_DeclinedReturnsErrNotConfirmed(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &leave.Options{
		RawRef: "@durov", IOStreams: ios, Prompter: &ui.StubPrompter{Answers: []any{false}}, Yes: false,
		Do: func(context.Context, actionchat.MembershipQuery) (output.ChatMembershipRow, error) {
			t.Fatal("Do must not run when prompt is declined")
			return output.ChatMembershipRow{}, nil
		},
	}
	err := leave.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestRun_YesBypassesPrompt(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &leave.Options{
		RawRef: "@durov", IOStreams: ios, Yes: true,
		Do: func(_ context.Context, args actionchat.MembershipQuery) (output.ChatMembershipRow, error) {
			return output.ChatMembershipRow{Action: "leave"}, nil
		},
	}
	require.NoError(t, leave.Run(context.Background(), opts))
}

func TestRun_InvalidRefIsUsage(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &leave.Options{
		RawRef: "", IOStreams: ios, Yes: true,
		Do: func(context.Context, actionchat.MembershipQuery) (output.ChatMembershipRow, error) {
			t.Fatal("Do must not run when ref invalid")
			return output.ChatMembershipRow{}, nil
		},
	}
	err := leave.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}
