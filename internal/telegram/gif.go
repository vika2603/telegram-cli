package telegram

import (
	"context"
	"errors"
	"fmt"

	gotdmessage "github.com/gotd/td/telegram/message"
	msgquery "github.com/gotd/td/telegram/query/messages"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"

	actionmessage "github.com/vika2603/telegram-cli/internal/action/message"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
)

// ListGifs performs the RPC for `tg msg gif list` (saved GIFs).
func ListGifs(ctx context.Context, api *tg.Client) ([]output.StickerRow, error) {
	res, err := api.MessagesGetSavedGifs(ctx, 0)
	if err != nil {
		return nil, err
	}
	full, ok := res.(*tg.MessagesSavedGifs)
	if !ok {
		return nil, nil
	}
	rows := make([]output.StickerRow, 0, len(full.Gifs))
	for _, dc := range full.Gifs {
		doc, ok := dc.AsNotEmpty()
		if !ok {
			continue
		}
		rows = append(rows, output.StickerRow{
			Kind: "gif",
			Ref:  actionmessage.EncodeGifToken(actionmessage.StickerDoc{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference}),
			ID:   doc.ID,
			Type: "gif",
		})
	}
	return rows, nil
}

// sendGif sends a saved GIF (by ref handle) or resends a gif from a message.
func sendGif(
	ctx context.Context,
	api *tg.Client,
	resolver *peer.Resolver,
	b *gotdmessage.RequestBuilder,
	src *actionmessage.StickerSource,
) (tg.UpdatesClass, error) {
	if src.Doc != nil {
		upd, err := b.Media(ctx, gotdmessage.Media(inputMediaFromStickerDoc(src.Doc)))
		// On an expired ref, refresh from the saved-gifs list and retry once.
		if err != nil && tgerr.Is(err, tg.ErrFileReferenceExpired, tg.ErrFileReferenceInvalid) {
			if fresh, rerr := refreshGifRef(ctx, api, src.Doc.ID); rerr == nil {
				return b.Media(ctx, gotdmessage.Media(inputMediaFromStickerDoc(fresh)))
			}
		}
		return upd, err
	}
	resolved, err := resolver.Resolve(ctx, src.Peer)
	if err != nil {
		return nil, err
	}
	elem, err := getMessageByID(ctx, api, resolved.InputPeer, src.MessageID)
	if err != nil {
		return nil, err
	}
	doc, ok := animatedInputDocument(elem)
	if !ok {
		return nil, fmt.Errorf("%w: %s is not a gif", command.ErrUsage, src.Peer.String())
	}
	return b.Media(ctx, gotdmessage.Media(&tg.InputMediaDocument{ID: doc}))
}

// animatedInputDocument extracts a resendable input document from a message,
// but only when the document is an animated gif.
func animatedInputDocument(elem msgquery.Elem) (*tg.InputDocument, bool) {
	msg, ok := elem.Msg.(*tg.Message)
	if !ok {
		return nil, false
	}
	media, ok := msg.Media.(*tg.MessageMediaDocument)
	if !ok {
		return nil, false
	}
	doc, ok := media.Document.AsNotEmpty()
	if !ok {
		return nil, false
	}
	for _, attr := range doc.Attributes {
		if _, ok := attr.(*tg.DocumentAttributeAnimated); ok {
			return doc.AsInput(), true
		}
	}
	return nil, false
}

// refreshGifRef re-fetches the saved-gifs list for a fresh file reference.
func refreshGifRef(ctx context.Context, api *tg.Client, id int64) (*actionmessage.StickerDoc, error) {
	res, err := api.MessagesGetSavedGifs(ctx, 0)
	if err != nil {
		return nil, err
	}
	full, ok := res.(*tg.MessagesSavedGifs)
	if !ok {
		return nil, errors.New("saved gifs unavailable")
	}
	for _, dc := range full.Gifs {
		if doc, ok := dc.AsNotEmpty(); ok && doc.ID == id {
			return &actionmessage.StickerDoc{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference}, nil
		}
	}
	return nil, errors.New("gif no longer saved")
}
