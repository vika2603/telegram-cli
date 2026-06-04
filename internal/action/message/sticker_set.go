package message

import (
	"context"
	"fmt"
	"strings"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// StickerSetRequest is the raw request for `tg msg sticker add` / `remove`.
type StickerSetRequest struct {
	RawSet string
	Remove bool
}

// StickerSetQuery is the normalized payload passed to the Telegram layer.
type StickerSetQuery struct {
	ShortName string
	Remove    bool
}

// StickerSetFunc installs or uninstalls a sticker set.
type StickerSetFunc func(context.Context, StickerSetQuery) (output.StickerSetResult, error)

// InstallStickerSet validates the request and dispatches the install/uninstall
// (uninstall when req.Remove). Named for the positive verb, mirroring
// FaveSticker, which likewise covers its inverse via a flag.
func InstallStickerSet(ctx context.Context, req StickerSetRequest, do StickerSetFunc) (output.StickerSetResult, error) {
	name, err := parseStickerSetRef(req.RawSet)
	if err != nil {
		return output.StickerSetResult{}, err
	}
	if do == nil {
		return output.StickerSetResult{}, fmt.Errorf("%w: msg sticker set called without function", command.ErrPrecondition)
	}
	return do(ctx, StickerSetQuery{ShortName: name, Remove: req.Remove})
}

// stickerSetURLPrefixes are the addstickers link forms a short name may be
// wrapped in; we strip them so the user can paste a share link directly.
var stickerSetURLPrefixes = []string{
	"https://t.me/addstickers/",
	"http://t.me/addstickers/",
	"https://telegram.me/addstickers/",
	"t.me/addstickers/",
	"telegram.me/addstickers/",
	"addstickers/",
}

// parseStickerSetRef accepts a sticker set short name or an addstickers link
// and returns the bare short name. Short-name character validation is left to
// Telegram, which rejects unknown sets with STICKERSET_INVALID.
func parseStickerSetRef(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	for _, p := range stickerSetURLPrefixes {
		if rest, ok := strings.CutPrefix(s, p); ok {
			s = rest
			break
		}
	}
	// A link may carry a trailing query/fragment (e.g. ?foo); keep only the
	// short-name segment.
	if i := strings.IndexAny(s, "/?#"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return "", fmt.Errorf("%w: empty sticker set reference", command.ErrUsage)
	}
	return s, nil
}
