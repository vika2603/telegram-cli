package telegram

import (
	"context"
	"errors"
	"fmt"

	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
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
		setID, setHash := stickerSet(doc)
		rows = append(rows, output.StickerRow{
			Kind: "sticker",
			Ref: actionmessage.EncodeStickerToken(actionmessage.StickerDoc{
				ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference,
				SetID: setID, SetAccessHash: setHash,
			}),
			ID:    doc.ID,
			Emoji: stickerEmoji(doc),
			Type:  stickerType(doc),
		})
	}
	return rows
}

// inputDocFromStickerDoc builds a tg input document from a sticker ref handle.
func inputDocFromStickerDoc(doc *actionmessage.StickerDoc) *tg.InputDocument {
	return &tg.InputDocument{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference}
}

// inputMediaFromStickerDoc builds the send media for a sticker ref handle.
func inputMediaFromStickerDoc(doc *actionmessage.StickerDoc) *tg.InputMediaDocument {
	return &tg.InputMediaDocument{ID: inputDocFromStickerDoc(doc)}
}

// resolveStickerInputDocument turns a sticker source (ref handle or message
// ref) into a sendable input document.
func resolveStickerInputDocument(ctx context.Context, api *tg.Client, resolver *peer.Resolver, src *actionmessage.StickerSource) (*tg.InputDocument, error) {
	if src.Doc != nil {
		return inputDocFromStickerDoc(src.Doc), nil
	}
	resolved, err := resolver.Resolve(ctx, src.Peer)
	if err != nil {
		return nil, err
	}
	elem, err := getMessageByID(ctx, api, resolved.InputPeer, src.MessageID)
	if err != nil {
		return nil, err
	}
	doc, ok := stickerInputDocument(elem)
	if !ok {
		return nil, fmt.Errorf("%w: %s is not a sticker", command.ErrUsage, src.Peer.String())
	}
	return doc, nil
}

// FaveSticker performs the RPC for `tg msg sticker fave` / `unfave`. On an
// expired ref it refreshes from the owning set and retries once.
func FaveSticker(ctx context.Context, api *tg.Client, resolver *peer.Resolver, q actionmessage.FaveQuery) (output.FaveResult, error) {
	doc, err := resolveStickerInputDocument(ctx, api, resolver, q.Source)
	if err != nil {
		return output.FaveResult{}, err
	}
	err = faveStickerCall(ctx, api, doc, q.Unfave)
	if err != nil && tgerr.Is(err, tg.ErrFileReferenceExpired, tg.ErrFileReferenceInvalid) && q.Source.Doc != nil && q.Source.Doc.SetID != 0 {
		if fresh, rerr := refreshStickerRef(ctx, api, q.Source.Doc); rerr == nil {
			doc = inputDocFromStickerDoc(fresh)
			err = faveStickerCall(ctx, api, doc, q.Unfave)
		}
	}
	if err != nil {
		return output.FaveResult{}, err
	}
	action := "fave"
	if q.Unfave {
		action = "unfave"
	}
	return output.FaveResult{Action: action, ID: doc.ID}, nil
}

func faveStickerCall(ctx context.Context, api *tg.Client, doc *tg.InputDocument, unfave bool) error {
	_, err := api.MessagesFaveSticker(ctx, &tg.MessagesFaveStickerRequest{ID: doc, Unfave: unfave})
	return err
}

// refreshStickerRef re-fetches the sticker's owning set and returns the doc
// with a fresh file reference. Used to recover from FILE_REFERENCE_EXPIRED.
func refreshStickerRef(ctx context.Context, api *tg.Client, doc *actionmessage.StickerDoc) (*actionmessage.StickerDoc, error) {
	set, err := api.MessagesGetStickerSet(ctx, &tg.MessagesGetStickerSetRequest{
		Stickerset: &tg.InputStickerSetID{ID: doc.SetID, AccessHash: doc.SetAccessHash},
	})
	if err != nil {
		return nil, err
	}
	full, ok := set.(*tg.MessagesStickerSet)
	if !ok {
		return nil, errors.New("sticker set unavailable")
	}
	for _, dc := range full.Documents {
		d, ok := dc.AsNotEmpty()
		if !ok || d.ID != doc.ID {
			continue
		}
		return &actionmessage.StickerDoc{
			ID: d.ID, AccessHash: d.AccessHash, FileReference: d.FileReference,
			SetID: doc.SetID, SetAccessHash: doc.SetAccessHash,
		}, nil
	}
	return nil, errors.New("sticker no longer in set")
}

// stickerSet returns the owning set's id and access hash, or (0, 0) if the
// sticker belongs to no set (so its file reference can't be set-refreshed).
func stickerSet(doc *tg.Document) (id, accessHash int64) {
	for _, attr := range doc.Attributes {
		s, ok := attr.(*tg.DocumentAttributeSticker)
		if !ok {
			continue
		}
		if set, ok := s.Stickerset.(*tg.InputStickerSetID); ok {
			return set.ID, set.AccessHash
		}
	}
	return 0, 0
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
