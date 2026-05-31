package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestPin_SetsPinnedTrue(t *testing.T) {
	row, err := actionchat.Pin(context.Background(), actionchat.PinRequest{
		RawRef: "@grp",
	}, func(_ context.Context, q actionchat.PinQuery) (output.ChatPinRow, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.True(t, q.Pinned)
		return output.ChatPinRow{Action: "pin", Pinned: q.Pinned}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "pin", row.Action)
	require.True(t, row.Pinned)
}

func TestUnpin_SetsPinnedFalse(t *testing.T) {
	row, err := actionchat.Unpin(context.Background(), actionchat.PinRequest{
		RawRef: "@grp",
	}, func(_ context.Context, q actionchat.PinQuery) (output.ChatPinRow, error) {
		require.False(t, q.Pinned)
		return output.ChatPinRow{Action: "unpin", Pinned: q.Pinned}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "unpin", row.Action)
	require.False(t, row.Pinned)
}

func TestPin_RejectsBadRef(t *testing.T) {
	_, err := actionchat.Pin(context.Background(), actionchat.PinRequest{
		RawRef: "",
	}, func(context.Context, actionchat.PinQuery) (output.ChatPinRow, error) {
		t.Fatal("pin must not run with an invalid ref")
		return output.ChatPinRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}
