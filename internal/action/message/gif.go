package message

import (
	"context"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// gifTokenPrefix marks a `msg gif list` ref handle, distinct from a sticker ref
// and from a message ref passed to `--gif`.
const gifTokenPrefix = "gif_"

// EncodeGifToken / DecodeGifToken mirror the sticker token but with a gif
// prefix. GIFs belong to no set, so the set fields stay zero.
func EncodeGifToken(doc StickerDoc) string { return encodeDocToken(gifTokenPrefix, doc) }

// DecodeGifToken reverses EncodeGifToken; ok is false when s isn't a gif token.
func DecodeGifToken(s string) (*StickerDoc, bool, error) { return decodeDocToken(gifTokenPrefix, s) }

// parseGifSource interprets a gif argument as a `msg gif list` ref token or a
// message ref.
func parseGifSource(raw string) (*StickerSource, error) {
	src := &StickerSource{}
	if doc, ok, err := DecodeGifToken(raw); err != nil {
		return nil, err
	} else if ok {
		src.Doc = doc
		return src, nil
	}
	mref, err := parseMessageRef(raw)
	if err != nil {
		return nil, err
	}
	src.Peer, src.MessageID = mref.Peer, mref.MessageID
	return src, nil
}

// normalizeGifSend handles `tg msg send <ref> --gif <ref>`. Like a sticker, a
// gif is sent on its own.
func normalizeGifSend(req SendRequest) (SendQuery, error) {
	if req.Text != "" {
		return SendQuery{}, fmt.Errorf("%w: --gif cannot be combined with message text", command.ErrUsage)
	}
	if len(compactStrings(req.Files)) > 0 {
		return SendQuery{}, fmt.Errorf("%w: --gif cannot be combined with --file", command.ErrUsage)
	}
	target, err := ref.ParseRef(req.RawRef)
	if err != nil {
		return SendQuery{}, fmt.Errorf("%w: %s", command.ErrUsage, err.Error())
	}
	source, err := parseGifSource(req.Gif)
	if err != nil {
		return SendQuery{}, err
	}
	return SendQuery{
		Ref:      target,
		Gif:      source,
		ReplyTo:  req.ReplyTo,
		Silent:   req.Silent,
		Schedule: req.Schedule,
		RandomID: req.RandomID,
	}, nil
}

// GifListFunc lists the account's saved GIFs.
type GifListFunc func(context.Context) ([]output.StickerRow, error)

// ListGifs dispatches `tg msg gif list`.
func ListGifs(ctx context.Context, do GifListFunc) ([]output.StickerRow, error) {
	if do == nil {
		return nil, fmt.Errorf("%w: msg gif list called without list function", command.ErrPrecondition)
	}
	return do(ctx)
}
