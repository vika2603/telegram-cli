package list_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/cli/msg/list"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_Flags(t *testing.T) {
	var captured *list.Options
	f := runtime.NewTestInvocation(t)
	cmd := list.New(f, func(o *list.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{
		"@ch",
		"--limit", "50",
		"--min-date", "2026-01-01T00:00:00Z",
		"--max-date", "2026-01-02T00:00:00Z",
		"--order", "asc",
	})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "@ch", captured.RawRef)
	require.Equal(t, 50, captured.Limit)
	require.Equal(t, "2026-01-01T00:00:00Z", captured.MinDate)
	require.Equal(t, "2026-01-02T00:00:00Z", captured.MaxDate)
	require.Equal(t, "asc", captured.Order)
}

func TestRun_BadDateIsUsage(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &list.Options{
		RawRef:    "@ch",
		Limit:     30,
		MinDate:   "not-a-date",
		IOStreams: ios,
	}
	err := list.Run(context.Background(), opts)
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_Renders(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &list.Options{
		RawRef:    "@ch",
		Limit:     30,
		IOStreams: ios,
		Fetch: func(_ context.Context, _ actionmessage.ListQuery) ([]output.MessageRow, error) {
			return []output.MessageRow{{ID: 1, Date: "2026-04-23T12:00:00Z", Text: "hi"}}, nil
		},
	}
	require.NoError(t, list.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "hi")
}

func TestRun_PassesFlagsToFetch(t *testing.T) {
	ios, _, _, _ := ui.Test()
	called := false
	opts := &list.Options{
		RawRef:    "@ch",
		Limit:     7,
		MinDate:   "2026-01-01T00:00:00Z",
		MaxDate:   "2026-01-02T00:00:00Z",
		Order:     "asc",
		IOStreams: ios,
		Fetch: func(_ context.Context, q actionmessage.ListQuery) ([]output.MessageRow, error) {
			called = true
			require.Equal(t, 7, q.Limit)
			require.True(t, q.Asc)
			require.Equal(t, "ch", q.Ref.Value)
			require.Equal(t, "2026-01-01T00:00:00Z", q.MinDate.Format("2006-01-02T15:04:05Z"))
			require.Equal(t, "2026-01-02T00:00:00Z", q.MaxDate.Format("2006-01-02T15:04:05Z"))
			return []output.MessageRow{{ID: 1, Date: "2026-01-01T00:00:00Z", Text: "hi"}}, nil
		},
	}
	require.NoError(t, list.Run(context.Background(), opts))
	require.True(t, called)
}

func TestRun_ExportsJSON(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &list.Options{
		RawRef:    "@ch",
		Limit:     30,
		IOStreams: ios,
		Exporter:  testExporter{},
		Fetch: func(_ context.Context, _ actionmessage.ListQuery) ([]output.MessageRow, error) {
			return []output.MessageRow{{ID: 1, Ref: "@ch:1", Date: "2026-04-23T12:00:00Z", Text: "hi"}}, nil
		},
	}
	require.NoError(t, list.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), `"ref":"@ch:1"`)
	require.Contains(t, stdout.String(), `"text":"hi"`)
}

type testExporter struct{}

func (testExporter) Fields() []string { return []string{"ref", "text"} }

func (testExporter) Write(io *ui.IOStreams, data any) error {
	return json.NewEncoder(io.Out).Encode(data)
}
