package edit_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/msg/edit"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *edit.Options
	f := runtime.NewTestInvocation(t)
	cmd := edit.New(f, func(o *edit.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"@alice:42", "--text", "fixed", "--parse", "html"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@alice:42", captured.RawMessageRef)
	require.Equal(t, "fixed", captured.Text)
	require.Equal(t, "html", captured.Parse)
}

func TestRun_EmptyTextRejected(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &edit.Options{
		RawMessageRef: "@a:1", IOStreams: ios,
		Edit: func(_ context.Context, _ actionmessage.EditQuery) (output.SendResultRow, error) {
			t.Fatal("Edit must not run when --text missing")
			return output.SendResultRow{}, nil
		},
	}
	err := edit.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_BadMessageRefRejected(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &edit.Options{
		RawMessageRef: "@a", Text: "hi", IOStreams: ios,
		Edit: func(_ context.Context, _ actionmessage.EditQuery) (output.SendResultRow, error) {
			t.Fatal("Edit must not run when message ref is invalid")
			return output.SendResultRow{}, nil
		},
	}
	err := edit.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_BadParseRejected(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &edit.Options{
		RawMessageRef: "@a:1", Text: "hi", Parse: "bogus", IOStreams: ios,
		Edit: func(_ context.Context, _ actionmessage.EditQuery) (output.SendResultRow, error) {
			t.Fatal("Edit must not run when --parse is invalid")
			return output.SendResultRow{}, nil
		},
	}
	err := edit.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_NilEditClosureReturnsPrecondition(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &edit.Options{RawMessageRef: "@a:1", Text: "hi", IOStreams: ios}
	err := edit.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StubbedEdit(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &edit.Options{
		RawMessageRef: "@a:77", Text: "hi",
		IOStreams: ios,
		Edit: func(_ context.Context, a actionmessage.EditQuery) (output.SendResultRow, error) {
			require.Equal(t, 77, a.MessageID)
			require.Equal(t, "a", a.Ref.Value)
			return output.SendResultRow{Action: "edit", MessageID: 77, ChatID: 10}, nil
		},
	}
	require.NoError(t, edit.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "edit")
	require.Contains(t, stdout.String(), "77")
}
