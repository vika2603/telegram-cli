package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionsearch "github.com/vika2603/telegram-cli/internal/action/search"
	"github.com/vika2603/telegram-cli/internal/cli/search/chat"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_Flags(t *testing.T) {
	var captured *chat.Options
	f := runtime.NewTestInvocation(t)
	cmd := chat.New(f, func(o *chat.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"durov", "--kind", "user", "--my-only", "--limit", "5"})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "durov", captured.Query)
	require.Equal(t, "user", captured.Kind)
	require.True(t, captured.MyOnly)
	require.Equal(t, 5, captured.Limit)
}

func TestRun_KindNarrowing(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &chat.Options{
		Query:     "x",
		Kind:      "user",
		Limit:     20,
		IOStreams: ios,
		Fetch: func(context.Context, actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
			return []output.SearchChatRow{
				{ChatRow: output.ChatRow{ID: 1, Kind: "user", Title: "Alice"}, Source: "my"},
				{ChatRow: output.ChatRow{ID: 2, Kind: "channel", Title: "News"}, Source: "public"},
			}, nil
		},
	}
	require.NoError(t, chat.Run(context.Background(), opts))
	got := stdout.String()
	require.Contains(t, got, "Alice")
	require.NotContains(t, got, "News", "channel should be filtered out by --kind user")
}

func TestRun_KindGroupMatchesChat(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &chat.Options{
		Query:     "x",
		Kind:      "group",
		Limit:     20,
		IOStreams: ios,
		Fetch: func(context.Context, actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
			return []output.SearchChatRow{
				{ChatRow: output.ChatRow{ID: 10, Kind: "chat", Title: "OurGroup"}, Source: "my"},
				{ChatRow: output.ChatRow{ID: 20, Kind: "channel", Title: "Feed"}, Source: "public"},
			}, nil
		},
	}
	require.NoError(t, chat.Run(context.Background(), opts))
	got := stdout.String()
	require.Contains(t, got, "OurGroup")
	require.NotContains(t, got, "Feed")
}

func TestRun_LimitCapsCombinedTotal(t *testing.T) {
	// ContactsSearch returns MyResults + Results which can exceed --limit
	// after our dedup merge. The user-facing --limit must cap the
	// final, post-filter row count.
	ios, _, stdout, _ := ui.Test()
	opts := &chat.Options{
		Query:     "telegram",
		Limit:     3,
		IOStreams: ios,
		Fetch: func(context.Context, actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
			return []output.SearchChatRow{
				{ChatRow: output.ChatRow{ID: 1, Kind: "user", Title: "KeptOne"}, Source: "my"},
				{ChatRow: output.ChatRow{ID: 2, Kind: "channel", Title: "KeptTwo"}, Source: "public"},
				{ChatRow: output.ChatRow{ID: 3, Kind: "channel", Title: "KeptThree"}, Source: "public"},
				{ChatRow: output.ChatRow{ID: 4, Kind: "channel", Title: "DroppedOverflow"}, Source: "public"},
			}, nil
		},
	}
	require.NoError(t, chat.Run(context.Background(), opts))
	got := stdout.String()
	require.Contains(t, got, "KeptOne")
	require.Contains(t, got, "KeptTwo")
	require.Contains(t, got, "KeptThree")
	require.NotContains(t, got, "DroppedOverflow", "rows beyond --limit must be dropped")
}

func TestRun_MyOnlyNarrowing(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	opts := &chat.Options{
		Query:     "x",
		MyOnly:    true,
		Limit:     20,
		IOStreams: ios,
		Fetch: func(context.Context, actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
			return []output.SearchChatRow{
				{ChatRow: output.ChatRow{ID: 1, Kind: "user", Title: "Mine"}, Source: "my"},
				{ChatRow: output.ChatRow{ID: 2, Kind: "user", Title: "Theirs"}, Source: "public"},
			}, nil
		},
	}
	require.NoError(t, chat.Run(context.Background(), opts))
	got := stdout.String()
	require.Contains(t, got, "Mine")
	require.NotContains(t, got, "Theirs")
}
