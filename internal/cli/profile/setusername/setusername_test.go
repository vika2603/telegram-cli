package setusername_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/profile/setusername"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *setusername.Options
	f := runtime.NewTestInvocation(t)
	cmd := setusername.New(f, func(o *setusername.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"newhandle"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "newhandle", captured.Username)
}

func TestNew_EmptyClears(t *testing.T) {
	var captured *setusername.Options
	f := runtime.NewTestInvocation(t)
	cmd := setusername.New(f, func(o *setusername.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{""})
	require.NoError(t, cmd.Execute())
	require.Empty(t, captured.Username)
}

func TestRun_NilUpdateClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &setusername.Options{Username: "foo", IOStreams: ios}
	err := setusername.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedUpdate(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &setusername.Options{
		Username: "foo", IOStreams: ios,
		Update: func(_ context.Context, s string) (output.ProfileRow, error) {
			require.Equal(t, "foo", s)
			return output.ProfileRow{Action: "set-username", Username: "foo"}, nil
		},
	}
	require.NoError(t, setusername.Run(context.Background(), opts))
	s := stdout.String()
	require.Contains(t, s, `"action":"set-username"`)
	require.Contains(t, s, `"username":"foo"`)
}

func TestRun_StubbedUpdateEmpty(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &setusername.Options{
		Username: "", IOStreams: ios,
		Update: func(_ context.Context, s string) (output.ProfileRow, error) {
			require.Empty(t, s)
			return output.ProfileRow{Action: "set-username"}, nil
		},
	}
	require.NoError(t, setusername.Run(context.Background(), opts))
}
