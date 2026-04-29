package message_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestNormalizeList_ClampsLimit(t *testing.T) {
	got, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef: "@ch",
		Limit:  5000,
		Order:  "desc",
	})
	require.NoError(t, err)
	require.Equal(t, 1000, got.Limit)
}

func TestNormalizeList_ParsesFlags(t *testing.T) {
	got, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef:  "@ch",
		Limit:   50,
		MinDate: "2026-01-01T00:00:00Z",
		MaxDate: "2026-01-02T00:00:00Z",
		Order:   "asc",
	})
	require.NoError(t, err)
	require.True(t, got.Asc)
	require.Equal(t, 50, got.Limit)
	require.Equal(t, "ch", got.Ref.Value)
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), got.MinDate)
	require.Equal(t, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC), got.MaxDate)
}

func TestNormalizeList_DefaultsToDescending(t *testing.T) {
	got, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef: "@ch",
		Limit:  30,
	})
	require.NoError(t, err)
	require.False(t, got.Asc)
}

func TestNormalizeList_RejectsBadLimit(t *testing.T) {
	_, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef: "@ch",
		Limit:  0,
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestNormalizeList_BadDateIsUsage(t *testing.T) {
	_, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef:  "@ch",
		Limit:   30,
		MinDate: "not-a-date",
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestNormalizeList_BadMaxDateIsUsage(t *testing.T) {
	_, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef:  "@ch",
		Limit:   30,
		MaxDate: "not-a-date",
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestNormalizeList_BadOrderIsUsage(t *testing.T) {
	_, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef: "@ch",
		Limit:  30,
		Order:  "oldest",
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestNormalizeList_PassesOffsetID(t *testing.T) {
	got, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef:   "@ch",
		Limit:    30,
		OffsetID: 1234,
	})
	require.NoError(t, err)
	require.Equal(t, 1234, got.OffsetID)
}

func TestNormalizeList_NegativeOffsetIDIsUsage(t *testing.T) {
	_, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef:   "@ch",
		Limit:    30,
		OffsetID: -1,
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestNormalizeList_PassesMinID(t *testing.T) {
	got, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef: "@ch",
		Limit:  30,
		MinID:  100,
	})
	require.NoError(t, err)
	require.Equal(t, 100, got.MinID)
}

func TestNormalizeList_NegativeMinIDIsUsage(t *testing.T) {
	_, err := actionmessage.NormalizeList(actionmessage.ListRequest{
		RawRef: "@ch",
		Limit:  30,
		MinID:  -1,
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestList_DelegatesValidatedQuery(t *testing.T) {
	rows, err := actionmessage.List(context.Background(), actionmessage.ListRequest{
		RawRef: "@ch",
		Limit:  30,
		Order:  "asc",
	}, func(_ context.Context, q actionmessage.ListQuery) ([]output.MessageRow, error) {
		require.True(t, q.Asc)
		require.Equal(t, 30, q.Limit)
		require.Equal(t, "ch", q.Ref.Value)
		return []output.MessageRow{{ID: 1, Text: "hi"}}, nil
	})
	require.NoError(t, err)
	require.Equal(t, []output.MessageRow{{ID: 1, Text: "hi"}}, rows)
}
