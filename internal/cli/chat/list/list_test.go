package list_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/cli/chat/list"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	var captured *list.Options
	f := runtime.NewTestInvocation(t)
	cmd := list.New(f, func(o *list.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"--limit", "50", "--archived"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, 50, captured.Limit)
	require.True(t, captured.ArchivedOnly)
}

func TestNew_DefaultLimit(t *testing.T) {
	var captured *list.Options
	f := runtime.NewTestInvocation(t)
	cmd := list.New(f, func(o *list.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())
	require.Equal(t, 30, captured.Limit)
}

func TestRun_RendersRows(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &list.Options{
		Limit:     30,
		IOStreams: ios,
		Fetch: func(context.Context, actionchat.ListRequest) ([]output.ChatRow, error) {
			return []output.ChatRow{
				{ID: 1, Kind: "user", Title: "Alice"},
				{ID: 2, Kind: "channel", Title: "News"},
			}, nil
		},
	}
	require.NoError(t, list.Run(context.Background(), opts))
	got := stdout.String()
	require.Contains(t, got, "Alice")
	require.Contains(t, got, "News")
}
