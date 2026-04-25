package setname_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/profile/setname"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *setname.Options
	f := runtime.NewTestInvocation(t)
	cmd := setname.New(f, func(o *setname.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"Bob", "--last", "Jones"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "Bob", captured.First)
	require.Equal(t, "Jones", captured.Last)
	require.True(t, captured.LastSet)
}

func TestNew_LastNotSetWhenOmitted(t *testing.T) {
	var captured *setname.Options
	f := runtime.NewTestInvocation(t)
	cmd := setname.New(f, func(o *setname.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"Bob"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "Bob", captured.First)
	require.Empty(t, captured.Last)
	require.False(t, captured.LastSet)
}

func TestNew_LastExplicitlyEmptyClears(t *testing.T) {
	var captured *setname.Options
	f := runtime.NewTestInvocation(t)
	cmd := setname.New(f, func(o *setname.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"Bob", "--last", ""})
	require.NoError(t, cmd.Execute())
	require.Empty(t, captured.Last)
	require.True(t, captured.LastSet)
}

func TestRun_NilUpdateClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &setname.Options{First: "Bob", IOStreams: ios}
	err := setname.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedUpdate(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &setname.Options{
		First: "Bob", Last: "Jones", LastSet: true, IOStreams: ios,
		Update: func(_ context.Context, a setname.UpdateArgs) (output.ProfileRow, error) {
			require.Equal(t, "Bob", a.First)
			require.Equal(t, "Jones", a.Last)
			require.True(t, a.LastSet)
			return output.ProfileRow{Action: "set-name", FirstName: "Bob", LastName: "Jones"}, nil
		},
	}
	require.NoError(t, setname.Run(context.Background(), opts))
	s := stdout.String()
	require.Contains(t, s, `"action":"set-name"`)
	require.Contains(t, s, `"first_name":"Bob"`)
	require.Contains(t, s, `"last_name":"Jones"`)
}

func TestRun_StubbedUpdateLastOmitted(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &setname.Options{
		First: "Bob", IOStreams: ios,
		Update: func(_ context.Context, a setname.UpdateArgs) (output.ProfileRow, error) {
			require.Equal(t, "Bob", a.First)
			require.Empty(t, a.Last)
			require.False(t, a.LastSet)
			return output.ProfileRow{Action: "set-name", FirstName: "Bob"}, nil
		},
	}
	require.NoError(t, setname.Run(context.Background(), opts))
}
