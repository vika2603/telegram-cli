package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

func TestShow_ParsesRef(t *testing.T) {
	row, err := actionchat.Show(context.Background(), actionchat.ShowRequest{RawRef: "@durov"},
		func(_ context.Context, q actionchat.ShowQuery) (output.ChatRow, error) {
			require.Equal(t, ref.Ref{Kind: ref.RefKindUsername, Value: "durov"}, q.Ref)
			return output.ChatRow{ID: 1, Kind: "user", Title: "Pavel"}, nil
		})

	require.NoError(t, err)
	require.Equal(t, int64(1), row.ID)
}

func TestShow_InvalidRef(t *testing.T) {
	_, err := actionchat.Show(context.Background(), actionchat.ShowRequest{RawRef: "not a ref"},
		func(context.Context, actionchat.ShowQuery) (output.ChatRow, error) {
			t.Fatal("fetch should not run")
			return output.ChatRow{}, nil
		})

	require.ErrorIs(t, err, command.ErrUsage)
}

func TestShow_RequiresFetch(t *testing.T) {
	_, err := actionchat.Show(context.Background(), actionchat.ShowRequest{RawRef: "@durov"}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}
