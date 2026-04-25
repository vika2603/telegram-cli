package join_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/join"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	var captured *join.Options
	cmd := join.New(f, func(o *join.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@durov"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, "@durov", captured.RawRef)
}

func TestRun_NilDoReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &join.Options{RawRef: "@durov", IOStreams: ios}
	err := join.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
	require.ErrorContains(t, err, "chat join called without do function")
}

func TestRun_InvalidRefIsUsage(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &join.Options{
		RawRef:    "",
		IOStreams: ios,
		Do: func(_ context.Context, _ actionchat.MembershipQuery) (output.ChatMembershipRow, error) {
			t.Fatal("Do must not be called when ref is invalid")
			return output.ChatMembershipRow{}, nil
		},
	}
	err := join.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_AlreadyMemberStubbed(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &join.Options{
		RawRef:    "@durov",
		IOStreams: ios,
		Do: func(_ context.Context, args actionchat.MembershipQuery) (output.ChatMembershipRow, error) {
			require.Equal(t, "durov", args.Ref.Value)
			return output.ChatMembershipRow{Action: "join", AlreadyMember: true}, nil
		},
	}
	require.NoError(t, join.Run(context.Background(), opts))
}

func TestRun_InviteLinkPassedToDo(t *testing.T) {
	ios, _, _, _ := ui.Test()
	var seen actionchat.MembershipQuery
	opts := &join.Options{
		RawRef:    "https://t.me/+abc123",
		IOStreams: ios,
		Do: func(_ context.Context, args actionchat.MembershipQuery) (output.ChatMembershipRow, error) {
			seen = args
			return output.ChatMembershipRow{Action: "join"}, nil
		},
	}
	require.NoError(t, join.Run(context.Background(), opts))
	require.True(t, seen.Ref.IsInviteLink(), "ref must be detected as invite link")
	require.Equal(t, "abc123", seen.Ref.InviteHash())
}
