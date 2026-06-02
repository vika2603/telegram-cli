package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// ListStickers performs the RPC for `tg msg sticker list`.
func ListStickers(ctx context.Context, api *tg.Client, q actionmessage.StickerListQuery) ([]output.StickerRow, error) {
	switch q.Source {
	case actionmessage.StickerRecent:
		return recentStickerRows(ctx, api)
	case actionmessage.StickerFaved:
		return favedStickerRows(ctx, api)
	case actionmessage.StickerInstalled:
		return installedStickerSetRows(ctx, api)
	case actionmessage.StickerAll:
		return allStickerRows(ctx, api)
	default:
		return nil, fmt.Errorf("%w: unknown sticker source %q", command.ErrUnsupported, q.Source)
	}
}

func recentStickerRows(ctx context.Context, api *tg.Client) ([]output.StickerRow, error) {
	res, err := api.MessagesGetRecentStickers(ctx, &tg.MessagesGetRecentStickersRequest{})
	if err != nil {
		return nil, err
	}
	full, ok := res.(*tg.MessagesRecentStickers)
	if !ok {
		return nil, nil
	}
	return stickerRowsFromDocs(full.Stickers), nil
}

func favedStickerRows(ctx context.Context, api *tg.Client) ([]output.StickerRow, error) {
	res, err := api.MessagesGetFavedStickers(ctx, 0)
	if err != nil {
		return nil, err
	}
	full, ok := res.(*tg.MessagesFavedStickers)
	if !ok {
		return nil, nil
	}
	return stickerRowsFromDocs(full.Stickers), nil
}

func installedStickerSetRows(ctx context.Context, api *tg.Client) ([]output.StickerRow, error) {
	res, err := api.MessagesGetAllStickers(ctx, 0)
	if err != nil {
		return nil, err
	}
	full, ok := res.(*tg.MessagesAllStickers)
	if !ok {
		return nil, nil
	}
	rows := make([]output.StickerRow, 0, len(full.Sets))
	for _, s := range full.Sets {
		rows = append(rows, output.StickerRow{Kind: "set", Set: s.ShortName, Title: s.Title, Count: s.Count})
	}
	return rows, nil
}

// allStickerRows returns every individual sticker available: recent + faved +
// each installed set expanded (one getStickerSet per set), deduplicated by
// document id. This is the slow, explicit path.
func allStickerRows(ctx context.Context, api *tg.Client) ([]output.StickerRow, error) {
	recent, err := recentStickerRows(ctx, api)
	if err != nil {
		return nil, err
	}
	faved, err := favedStickerRows(ctx, api)
	if err != nil {
		return nil, err
	}
	rows := make([]output.StickerRow, 0, len(recent)+len(faved))
	rows = append(rows, recent...)
	rows = append(rows, faved...)

	all, err := api.MessagesGetAllStickers(ctx, 0)
	if err != nil {
		return nil, err
	}
	if full, ok := all.(*tg.MessagesAllStickers); ok {
		for _, s := range full.Sets {
			set, err := api.MessagesGetStickerSet(ctx, &tg.MessagesGetStickerSetRequest{
				Stickerset: &tg.InputStickerSetID{ID: s.ID, AccessHash: s.AccessHash},
			})
			if err != nil {
				return nil, err
			}
			if full, ok := set.(*tg.MessagesStickerSet); ok {
				rows = append(rows, stickerRowsFromDocs(full.Documents)...)
			}
		}
	}
	return dedupStickerRows(rows), nil
}

func dedupStickerRows(rows []output.StickerRow) []output.StickerRow {
	seen := make(map[int64]struct{}, len(rows))
	out := rows[:0]
	for _, r := range rows {
		if _, dup := seen[r.ID]; dup {
			continue
		}
		seen[r.ID] = struct{}{}
		out = append(out, r)
	}
	return out
}

func stickerRowsFromDocs(docs []tg.DocumentClass) []output.StickerRow {
	rows := make([]output.StickerRow, 0, len(docs))
	for _, dc := range docs {
		doc, ok := dc.AsNotEmpty()
		if !ok {
			continue
		}
		rows = append(rows, output.StickerRow{
			Kind:  "sticker",
			Ref:   actionmessage.EncodeStickerToken(actionmessage.StickerDoc{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference}),
			ID:    doc.ID,
			Emoji: stickerEmoji(doc),
			Type:  stickerType(doc),
		})
	}
	return rows
}

func stickerEmoji(doc *tg.Document) string {
	for _, attr := range doc.Attributes {
		if s, ok := attr.(*tg.DocumentAttributeSticker); ok {
			return s.Alt
		}
	}
	return ""
}

func stickerType(doc *tg.Document) string {
	switch doc.MimeType {
	case "application/x-tgsticker":
		return "animated"
	case "video/webm":
		return "video"
	default:
		return "static"
	}
}
