package search_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	actionsearch "github.com/vika2603/telegram-cli/internal/action/search"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestNormalizeMessage_ClampsLimitAndParsesFlags(t *testing.T) {
	got, err := actionsearch.NormalizeMessage(actionsearch.MessageRequest{
		Query:   "hello",
		In:      "@chat",
		From:    "@ada",
		Filter:  "photos",
		MinDate: "2026-04-24T10:00:00Z",
		MaxDate: "2026-04-24T12:00:00Z",
		Limit:   5000,
		Order:   "asc",
	})
	require.NoError(t, err)
	require.Equal(t, "hello", got.Query)
	require.Equal(t, 1000, got.Limit)
	require.True(t, got.Asc)
	require.Equal(t, actionsearch.MessageFilterPhotos, got.Filter)
	require.Equal(t, "chat", got.InRef.Value)
	require.Equal(t, "ada", got.FromRef.Value)
	minDate, err := time.Parse(time.RFC3339, "2026-04-24T10:00:00Z")
	require.NoError(t, err)
	maxDate, err := time.Parse(time.RFC3339, "2026-04-24T12:00:00Z")
	require.NoError(t, err)
	require.Equal(t, minDate, got.MinDate)
	require.Equal(t, maxDate, got.MaxDate)
}

func TestNormalizeMessage_InvalidFilterIsUsage(t *testing.T) {
	_, err := actionsearch.NormalizeMessage(actionsearch.MessageRequest{
		Query:  "hi",
		Filter: "bogus",
		Limit:  30,
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestNormalizeMessage_FromRequiresIn(t *testing.T) {
	_, err := actionsearch.NormalizeMessage(actionsearch.MessageRequest{
		Query: "hi",
		From:  "@ada",
		Limit: 30,
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestNormalizeMessage_OnlyOneGlobalOriginFilter(t *testing.T) {
	_, err := actionsearch.NormalizeMessage(actionsearch.MessageRequest{
		Query:          "hi",
		BroadcastsOnly: true,
		UsersOnly:      true,
		Limit:          30,
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestMessage_DelegatesValidatedQuery(t *testing.T) {
	rows, err := actionsearch.Message(context.Background(), actionsearch.MessageRequest{
		Query:     "hi",
		Filter:    "phone-calls",
		Missed:    true,
		UsersOnly: true,
		Limit:     30,
		Order:     "desc",
	}, func(_ context.Context, q actionsearch.MessageQuery) ([]output.SearchMsgRow, error) {
		require.Equal(t, "hi", q.Query)
		require.Equal(t, actionsearch.MessageFilterPhoneCalls, q.Filter)
		require.True(t, q.Missed)
		require.True(t, q.UsersOnly)
		require.False(t, q.Asc)
		return []output.SearchMsgRow{{MessageID: 1, Text: "hi"}}, nil
	})
	require.NoError(t, err)
	require.Equal(t, []output.SearchMsgRow{{MessageID: 1, Text: "hi"}}, rows)
}
