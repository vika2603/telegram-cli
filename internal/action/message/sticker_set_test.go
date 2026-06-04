package message_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestStickerSet_ParsesShortNameAndLinks(t *testing.T) {
	for _, in := range []string{
		"Animals",
		"addstickers/Animals",
		"t.me/addstickers/Animals",
		"https://t.me/addstickers/Animals",
		"https://telegram.me/addstickers/Animals",
		"https://t.me/addstickers/Animals?utm=x",
	} {
		_, err := actionmessage.StickerSet(context.Background(), actionmessage.StickerSetRequest{RawSet: in},
			func(_ context.Context, q actionmessage.StickerSetQuery) (output.StickerSetResult, error) {
				require.Equal(t, "Animals", q.ShortName, in)
				require.False(t, q.Remove)
				return output.StickerSetResult{Action: "add", Set: q.ShortName}, nil
			})
		require.NoError(t, err, in)
	}
}

func TestStickerSet_RemovePassesThrough(t *testing.T) {
	_, err := actionmessage.StickerSet(context.Background(), actionmessage.StickerSetRequest{RawSet: "Animals", Remove: true},
		func(_ context.Context, q actionmessage.StickerSetQuery) (output.StickerSetResult, error) {
			require.Equal(t, "Animals", q.ShortName)
			require.True(t, q.Remove)
			return output.StickerSetResult{Action: "remove", Set: q.ShortName}, nil
		})
	require.NoError(t, err)
}

func TestStickerSet_RejectsEmptyRef(t *testing.T) {
	_, err := actionmessage.StickerSet(context.Background(), actionmessage.StickerSetRequest{RawSet: "  "},
		func(context.Context, actionmessage.StickerSetQuery) (output.StickerSetResult, error) {
			t.Fatal("must not dispatch")
			return output.StickerSetResult{}, nil
		})
	require.ErrorIs(t, err, command.ErrUsage)
}
