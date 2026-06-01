package chat_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionchat "github.com/vika2603/telegram-cli/internal/action/chat"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestSetPhoto_ParsesRefAndPath(t *testing.T) {
	row, err := actionchat.SetPhoto(context.Background(), actionchat.PhotoRequest{
		RawRef: "@grp",
		Path:   "/tmp/a.png",
	}, func(_ context.Context, q actionchat.PhotoQuery) (output.ChatPhotoRow, error) {
		require.Equal(t, "grp", q.Ref.Value)
		require.Equal(t, "/tmp/a.png", q.Path)
		require.False(t, q.Clear)
		return output.ChatPhotoRow{Action: "set"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "set", row.Action)
}

func TestSetPhoto_RequiresPath(t *testing.T) {
	_, err := actionchat.SetPhoto(context.Background(), actionchat.PhotoRequest{
		RawRef: "@grp",
	}, func(context.Context, actionchat.PhotoQuery) (output.ChatPhotoRow, error) {
		t.Fatal("must not dispatch without a path")
		return output.ChatPhotoRow{}, nil
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestClearPhoto_SetsClear(t *testing.T) {
	row, err := actionchat.ClearPhoto(context.Background(), actionchat.PhotoRequest{
		RawRef: "@grp",
	}, func(_ context.Context, q actionchat.PhotoQuery) (output.ChatPhotoRow, error) {
		require.True(t, q.Clear)
		return output.ChatPhotoRow{Action: "clear"}, nil
	})
	require.NoError(t, err)
	require.Equal(t, "clear", row.Action)
}

func TestSetPhoto_NilDoReturnsPrecondition(t *testing.T) {
	_, err := actionchat.SetPhoto(context.Background(), actionchat.PhotoRequest{RawRef: "@grp", Path: "/tmp/a.png"}, nil)
	require.ErrorIs(t, err, command.ErrPrecondition)
}
