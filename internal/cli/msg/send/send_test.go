package send_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/msg/send"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *send.Options
	f := runtime.NewTestInvocation(t)
	cmd := send.New(f, func(o *send.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{
		"@alice",
		"hello",
		"world",
		"--reply-to", "42",
		"--silent",
		"--schedule", "2026-04-25T09:00:00Z",
		"--parse", "html",
		"--file", "/tmp/x.pdf",
		"--file", "/tmp/y.pdf",
		"--name", "x-renamed.pdf",
		"--name", "y-renamed.pdf",
	})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, "@alice", captured.RawRef)
	require.Equal(t, "hello world", captured.Text)
	require.Equal(t, 42, captured.ReplyTo)
	require.True(t, captured.Silent)
	want, _ := time.Parse(time.RFC3339, "2026-04-25T09:00:00Z")
	require.Equal(t, want, captured.Schedule)
	require.Equal(t, "html", captured.Parse)
	require.Equal(t, []string{"/tmp/x.pdf", "/tmp/y.pdf"}, captured.Files)
	require.Equal(t, []string{"x-renamed.pdf", "y-renamed.pdf"}, captured.Names)
}

func TestNew_RequiresPeer(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := send.New(f, nil)
	cmd.SetArgs([]string{"hi"})
	err := cmd.Execute()
	require.Error(t, err)
}

func TestRun_StubbedSendRendersTTY(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &send.Options{
		RawRef:    "@alice",
		Text:      "hello",
		IOStreams: ios,
		Send: func(_ context.Context, a actionmessage.SendQuery) ([]output.SendResultRow, error) {
			require.Equal(t, "hello", a.Text)
			require.Equal(t, "alice", a.Ref.Value)
			return []output.SendResultRow{{Action: "send", MessageID: 1, ChatID: 10, Date: "2026-04-24T00:00:00Z"}}, nil
		},
	}
	require.NoError(t, send.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "send")
	require.Contains(t, stdout.String(), "1")
}

func TestRun_ValidationErrors(t *testing.T) {
	// missing text and file
	opts := &send.Options{RawRef: "@alice", IOStreams: newTestIO(t)}
	err := send.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)

	// bad parse mode
	opts2 := &send.Options{RawRef: "@alice", Text: "hi", Parse: "bogus", IOStreams: newTestIO(t)}
	err = send.Run(context.Background(), opts2)
	require.ErrorIs(t, err, command.ErrUsage)

	// text '-' and --file - both read stdin
	opts4 := &send.Options{RawRef: "@alice", Text: "-", Files: []string{"-"}, IOStreams: newTestIO(t)}
	err = send.Run(context.Background(), opts4)
	require.ErrorIs(t, err, command.ErrUsage)
}

func newTestIO(t *testing.T) *ui.IOStreams {
	t.Helper()
	ios, _, _, _ := ui.Test()
	return ios
}
