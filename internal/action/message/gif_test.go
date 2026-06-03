package message_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

func TestGifToken_RoundTrip(t *testing.T) {
	in := actionmessage.StickerDoc{ID: 99, AccessHash: -7, FileReference: []byte{0x09, 0x08}}
	tok := actionmessage.EncodeGifToken(in)
	out, ok, err := actionmessage.DecodeGifToken(tok)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, in.ID, out.ID)
	require.Equal(t, in.FileReference, out.FileReference)
}

func TestGifToken_NotConfusedWithSticker(t *testing.T) {
	gifTok := actionmessage.EncodeGifToken(actionmessage.StickerDoc{ID: 1, FileReference: []byte{1}})
	// A gif token must not decode as a sticker token, and vice versa.
	_, ok, err := actionmessage.DecodeStickerToken(gifTok)
	require.NoError(t, err)
	require.False(t, ok)

	stkTok := actionmessage.EncodeStickerToken(actionmessage.StickerDoc{ID: 1, FileReference: []byte{1}})
	_, ok, err = actionmessage.DecodeGifToken(stkTok)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestSend_GifTokenResolvesToDoc(t *testing.T) {
	tok := actionmessage.EncodeGifToken(actionmessage.StickerDoc{ID: 5, AccessHash: 6, FileReference: []byte{0xbb}})
	_, err := actionmessage.Send(context.Background(), actionmessage.SendRequest{
		RawRef: "@bob", Gif: tok,
	}, func(_ context.Context, q actionmessage.SendQuery) ([]output.SendResultRow, error) {
		require.NotNil(t, q.Gif)
		require.NotNil(t, q.Gif.Doc)
		require.Equal(t, int64(5), q.Gif.Doc.ID)
		require.Nil(t, q.Sticker)
		return []output.SendResultRow{{Action: "send", MessageID: 1}}, nil
	})
	require.NoError(t, err)
}

func TestSend_GifMsgRef(t *testing.T) {
	_, err := actionmessage.Send(context.Background(), actionmessage.SendRequest{
		RawRef: "@bob", Gif: "@alice:42",
	}, func(_ context.Context, q actionmessage.SendQuery) ([]output.SendResultRow, error) {
		require.NotNil(t, q.Gif)
		require.Equal(t, "alice", q.Gif.Peer.Value)
		require.Equal(t, 42, q.Gif.MessageID)
		return []output.SendResultRow{{Action: "send"}}, nil
	})
	require.NoError(t, err)
}

func TestSend_GifRejectsTextAndFile(t *testing.T) {
	for _, req := range []actionmessage.SendRequest{
		{RawRef: "@bob", Gif: "@a:1", Text: "hi"},
		{RawRef: "@bob", Gif: "@a:1", Files: []string{"/tmp/x"}},
		{RawRef: "@bob", Gif: "@a:1", Sticker: "@a:2"},
	} {
		_, err := actionmessage.Send(context.Background(), req,
			func(context.Context, actionmessage.SendQuery) ([]output.SendResultRow, error) {
				t.Fatal("must not dispatch")
				return nil, nil
			})
		require.ErrorIs(t, err, command.ErrUsage)
	}
}
