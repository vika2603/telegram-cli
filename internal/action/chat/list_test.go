package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestList_RejectsNonPositiveLimit(t *testing.T) {
	_, err := actionchat.List(context.Background(), actionchat.ListRequest{Limit: 0}, func(context.Context, actionchat.ListRequest) ([]output.ChatRow, error) {
		t.Fatal("fetch must not run")
		return nil, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestList_ClampsLimit(t *testing.T) {
	var got actionchat.ListRequest
	rows, err := actionchat.List(context.Background(), actionchat.ListRequest{Limit: 2000}, func(_ context.Context, req actionchat.ListRequest) ([]output.ChatRow, error) {
		got = req
		return []output.ChatRow{{ID: 1, Kind: "user"}}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 1000, got.Limit)
	require.Len(t, rows, 1)
}
