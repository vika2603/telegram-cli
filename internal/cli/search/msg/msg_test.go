package msg_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionsearch "github.com/vika2603/telegram-cli/internal/action/search"
	"github.com/vika2603/telegram-cli/internal/cli/search/msg"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_Flags(t *testing.T) {
	var captured *msg.Options
	f := runtime.NewTestInvocation(t)
	cmd := msg.New(f, func(o *msg.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"hello", "--in", "@chat", "--filter", "photos", "--limit", "50"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "hello", captured.Query)
	require.Equal(t, "@chat", captured.In)
	require.Equal(t, "photos", captured.Filter)
	require.Equal(t, 50, captured.Limit)
}

func TestRun_InvalidFilterIsUsage(t *testing.T) {
	ios, _, _, _ := ui.Test()
	opts := &msg.Options{
		Query:     "hi",
		Filter:    "bogus",
		Limit:     30,
		IOStreams: ios,
	}
	err := msg.Run(context.Background(), opts)
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_Renders(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &msg.Options{
		Query:     "hi",
		Limit:     30,
		IOStreams: ios,
		Fetch: func(context.Context, actionsearch.MessageQuery) ([]output.SearchMsgRow, error) {
			return []output.SearchMsgRow{
				{MessageID: 1, ChatID: 100, ChatTitle: "News", Date: "2026-04-23T12:00:00Z", Text: "hi"},
			}, nil
		},
	}
	require.NoError(t, msg.Run(context.Background(), opts))
	require.Contains(t, stdout.String(), "News")
}
