package unblock_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/cli/contact/unblock"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *unblock.Options
	f := runtime.NewTestInvocation(t)
	cmd := unblock.New(f, func(o *unblock.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"@bob"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@bob", captured.RawRef)
}

func TestRun_NilUnblockClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &unblock.Options{RawRef: "@bob", IOStreams: ios}
	err := unblock.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedUnblock(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	called := false
	opts := &unblock.Options{
		RawRef: "@bob", IOStreams: ios,
		Unblock: func(_ context.Context, q actioncontact.PeerQuery) error {
			called = true
			require.Equal(t, "bob", q.Ref.Value)
			return nil
		},
	}
	require.NoError(t, unblock.Run(context.Background(), opts))
	require.True(t, called)
	require.Equal(t, "unblocked\t@bob\n", stdout.String())
}
