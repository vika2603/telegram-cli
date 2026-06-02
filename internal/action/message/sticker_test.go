package message_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestStickerToken_RoundTrip(t *testing.T) {
	in := actionmessage.StickerDoc{
		ID: 12345, AccessHash: -98765, FileReference: []byte{0x01, 0x02, 0xff, 0x00},
		SetID: 555, SetAccessHash: -777,
	}
	tok := actionmessage.EncodeStickerToken(in)
	require.Greater(t, len(tok), 4)

	out, ok, err := actionmessage.DecodeStickerToken(tok)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, in.ID, out.ID)
	require.Equal(t, in.AccessHash, out.AccessHash)
	require.Equal(t, in.FileReference, out.FileReference)
	require.Equal(t, in.SetID, out.SetID)
	require.Equal(t, in.SetAccessHash, out.SetAccessHash)
}

func TestStickerToken_NotATokenFallsThrough(t *testing.T) {
	// A message ref must not be mistaken for a sticker token.
	_, ok, err := actionmessage.DecodeStickerToken("@alice:42")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestStickerToken_MalformedErrors(t *testing.T) {
	_, _, err := actionmessage.DecodeStickerToken("stk_!!!not-base64!!!")
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestSend_StickerTokenResolvesToDoc(t *testing.T) {
	tok := actionmessage.EncodeStickerToken(actionmessage.StickerDoc{ID: 7, AccessHash: 9, FileReference: []byte{0xaa}})
	_, err := actionmessage.Send(context.Background(), actionmessage.SendRequest{
		RawRef:  "@bob",
		Sticker: tok,
	}, func(_ context.Context, q actionmessage.SendQuery) ([]output.SendResultRow, error) {
		require.NotNil(t, q.Sticker)
		require.NotNil(t, q.Sticker.Doc)
		require.Equal(t, int64(7), q.Sticker.Doc.ID)
		require.Zero(t, q.Sticker.MessageID)
		return []output.SendResultRow{{Action: "send", MessageID: 1}}, nil
	})
	require.NoError(t, err)
}

func TestListStickers_RejectsUnknownSource(t *testing.T) {
	_, err := actionmessage.ListStickers(context.Background(), actionmessage.StickerListRequest{Source: "bogus"},
		func(context.Context, actionmessage.StickerListQuery) ([]output.StickerRow, error) {
			t.Fatal("must not dispatch")
			return nil, nil
		})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestListStickers_DefaultsToRecent(t *testing.T) {
	_, err := actionmessage.ListStickers(context.Background(), actionmessage.StickerListRequest{},
		func(_ context.Context, q actionmessage.StickerListQuery) ([]output.StickerRow, error) {
			require.Equal(t, actionmessage.StickerRecent, q.Source)
			return nil, nil
		})
	require.NoError(t, err)
}
