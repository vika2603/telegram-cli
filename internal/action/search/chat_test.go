package search_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionsearch "github.com/vika2603/telegram-cli/internal/action/search"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestChat_ValidatesKind(t *testing.T) {
	_, err := actionsearch.Chat(context.Background(), actionsearch.ChatRequest{Query: "x", Kind: "space", Limit: 20},
		func(context.Context, actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
			t.Fatal("fetch should not run")
			return nil, nil
		})

	require.ErrorIs(t, err, command.ErrUsage)
}

func TestChat_ValidatesLimit(t *testing.T) {
	_, err := actionsearch.Chat(context.Background(), actionsearch.ChatRequest{Query: "x", Limit: 0},
		func(context.Context, actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
			t.Fatal("fetch should not run")
			return nil, nil
		})

	require.ErrorIs(t, err, command.ErrUsage)
}

func TestChat_FiltersAndCapsRows(t *testing.T) {
	rows, err := actionsearch.Chat(context.Background(), actionsearch.ChatRequest{
		Query:  "telegram",
		Kind:   "group",
		MyOnly: true,
		Limit:  2,
	}, func(_ context.Context, q actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
		require.Equal(t, "telegram", q.Query)
		require.Equal(t, 2, q.Limit)
		return []output.SearchChatRow{
			{ChatRow: output.ChatRow{ID: 1, Kind: "chat", Title: "KeptOne"}, Source: "my"},
			{ChatRow: output.ChatRow{ID: 2, Kind: "channel", Title: "DroppedKind"}, Source: "my"},
			{ChatRow: output.ChatRow{ID: 3, Kind: "chat", Title: "DroppedSource"}, Source: "public"},
			{ChatRow: output.ChatRow{ID: 4, Kind: "chat", Title: "KeptTwo"}, Source: "my"},
			{ChatRow: output.ChatRow{ID: 5, Kind: "chat", Title: "DroppedLimit"}, Source: "my"},
		}, nil
	})

	require.NoError(t, err)
	require.Equal(t, []output.SearchChatRow{
		{ChatRow: output.ChatRow{ID: 1, Kind: "chat", Title: "KeptOne"}, Source: "my"},
		{ChatRow: output.ChatRow{ID: 4, Kind: "chat", Title: "KeptTwo"}, Source: "my"},
	}, rows)
}

func TestChat_ClampsLimit(t *testing.T) {
	_, err := actionsearch.Chat(context.Background(), actionsearch.ChatRequest{Query: "x", Limit: 2000},
		func(_ context.Context, q actionsearch.ChatQuery) ([]output.SearchChatRow, error) {
			require.Equal(t, 1000, q.Limit)
			return nil, nil
		})

	require.NoError(t, err)
}

func TestChat_RequiresFetch(t *testing.T) {
	_, err := actionsearch.Chat(context.Background(), actionsearch.ChatRequest{Query: "x", Limit: 20}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}
