package message

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// stickerTokenPrefix marks a `msg sticker list` ref handle, distinguishing it
// from a message ref passed to `--sticker`.
const stickerTokenPrefix = "stk_"

// EncodeStickerToken packs a sticker's input-document triple into a
// self-contained, copy-pasteable handle. The file reference is short-lived, so
// a token only stays valid for a while after listing.
func EncodeStickerToken(doc StickerDoc) string {
	buf := make([]byte, 32+len(doc.FileReference))
	binary.BigEndian.PutUint64(buf[0:8], uint64(doc.ID))
	binary.BigEndian.PutUint64(buf[8:16], uint64(doc.AccessHash))
	binary.BigEndian.PutUint64(buf[16:24], uint64(doc.SetID))
	binary.BigEndian.PutUint64(buf[24:32], uint64(doc.SetAccessHash))
	copy(buf[32:], doc.FileReference)
	return stickerTokenPrefix + base64.RawURLEncoding.EncodeToString(buf)
}

// DecodeStickerToken reverses EncodeStickerToken. ok is false when s is not a
// sticker token (so the caller can fall back to message-ref parsing); err is
// returned only when s looks like a token but is malformed.
func DecodeStickerToken(s string) (*StickerDoc, bool, error) {
	if len(s) <= len(stickerTokenPrefix) || s[:len(stickerTokenPrefix)] != stickerTokenPrefix {
		return nil, false, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s[len(stickerTokenPrefix):])
	if err != nil || len(raw) < 32 {
		return nil, false, fmt.Errorf("%w: invalid sticker ref %q", command.ErrUsage, s)
	}
	return &StickerDoc{
		ID:            int64(binary.BigEndian.Uint64(raw[0:8])),
		AccessHash:    int64(binary.BigEndian.Uint64(raw[8:16])),
		SetID:         int64(binary.BigEndian.Uint64(raw[16:24])),
		SetAccessHash: int64(binary.BigEndian.Uint64(raw[24:32])),
		FileReference: raw[32:],
	}, true, nil
}

// StickerListSource selects which sticker collection to list.
type StickerListSource string

const (
	StickerRecent    StickerListSource = "recent"
	StickerFaved     StickerListSource = "faved"
	StickerInstalled StickerListSource = "installed"
	// StickerAll is every individual sticker available to the account: recent
	// + favorited + every installed set expanded, deduplicated. It fetches each
	// set, so it is the slow, explicit option.
	StickerAll StickerListSource = "all"
)

// StickerListRequest is the raw request for `tg msg sticker list`.
type StickerListRequest struct {
	Source StickerListSource
}

// StickerListQuery is the normalized payload passed to Telegram.
type StickerListQuery struct {
	Source StickerListSource
}

// StickerListFunc lists stickers from the chosen source.
type StickerListFunc func(context.Context, StickerListQuery) ([]output.StickerRow, error)

// ListStickers validates and dispatches `tg msg sticker list`.
func ListStickers(ctx context.Context, req StickerListRequest, do StickerListFunc) ([]output.StickerRow, error) {
	switch req.Source {
	case StickerRecent, StickerFaved, StickerInstalled, StickerAll:
	case "":
		req.Source = StickerRecent
	default:
		return nil, fmt.Errorf("%w: unknown sticker source %q", command.ErrUsage, req.Source)
	}
	if do == nil {
		return nil, fmt.Errorf("%w: msg sticker list called without list function", command.ErrPrecondition)
	}
	return do(ctx, StickerListQuery(req))
}
