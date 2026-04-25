package add_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actioncontact "github.com/vika2603/telegram-cli/internal/action/contact"
	"github.com/vika2603/telegram-cli/internal/cli/contact/add"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *add.Options
	f := runtime.NewTestInvocation(t)
	cmd := add.New(f, func(o *add.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{"+15551234", "--first", "Alice", "--last", "Smith", "--mutual"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "+15551234", captured.Phone)
	require.Equal(t, "Alice", captured.First)
	require.Equal(t, "Smith", captured.Last)
	require.True(t, captured.Mutual)
}

func TestRun_NilAddClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &add.Options{Phone: "+1", First: "Alice", IOStreams: ios}
	err := add.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_RequiresFirst(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &add.Options{
		Phone: "+1", IOStreams: ios,
		Add: func(_ context.Context, _ actioncontact.AddQuery) (output.ContactRow, error) {
			t.Fatal("Add must not be called when --first missing")
			return output.ContactRow{}, nil
		},
	}
	err := add.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_StubbedAdd(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &add.Options{
		Phone: "+15551234", First: "Alice", Last: "Smith", Mutual: true, IOStreams: ios,
		Add: func(_ context.Context, a actioncontact.AddQuery) (output.ContactRow, error) {
			require.Equal(t, "+15551234", a.Phone)
			require.Equal(t, "Alice", a.First)
			require.Equal(t, "Smith", a.Last)
			require.True(t, a.Mutual)
			return output.ContactRow{ID: 10, FirstName: "Alice", LastName: "Smith", Phone: "+15551234", Mutual: true}, nil
		},
	}
	require.NoError(t, add.Run(context.Background(), opts))
	s := stdout.String()
	require.Contains(t, s, "Alice")
	require.Contains(t, s, "10")
}
