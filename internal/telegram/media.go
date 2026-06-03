package telegram

import (
	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ref"
)

// albumItemsDetailed builds album members with full media detail (for msg info).
func albumItemsDetailed(members []*tg.Message, baseRef string) []output.AlbumItem {
	items := make([]output.AlbumItem, 0, len(members))
	for _, m := range members {
		it := output.AlbumItem{ID: m.ID, Ref: ref.FormatMessageRef(baseRef, m.ID), Text: m.Message}
		if media, ok := m.GetMedia(); ok && media != nil {
			it.Media = mediaDetail(media)
		}
		items = append(items, it)
	}
	return items
}

// mediaDetail builds a full media object (filename, size, dimensions, etc.)
// from a message's media. Used by `msg info`; `msg list` keeps only the kind.
func mediaDetail(media tg.MessageMediaClass) *output.MediaObject {
	switch v := media.(type) {
	case *tg.MessageMediaPhoto:
		o := &output.MediaObject{Type: "photo"}
		if ph, ok := v.Photo.AsNotEmpty(); ok {
			o.Width, o.Height = largestPhotoSize(ph)
		}
		return o
	case *tg.MessageMediaDocument:
		doc, ok := v.Document.AsNotEmpty()
		if !ok {
			return &output.MediaObject{Type: "document"}
		}
		return documentDetail(doc)
	case *tg.MessageMediaWebPage:
		o := &output.MediaObject{Type: "web_page"}
		if wp, ok := v.Webpage.(*tg.WebPage); ok {
			o.URL = wp.URL
			o.Title, _ = wp.GetTitle()
			o.SiteName, _ = wp.GetSiteName()
			o.Description, _ = wp.GetDescription()
		}
		return o
	case *tg.MessageMediaPoll:
		pi := pollInfoFromMedia(v)
		return &output.MediaObject{Type: "poll", Poll: &pi}
	case *tg.MessageMediaGeo, *tg.MessageMediaGeoLive, *tg.MessageMediaVenue:
		return &output.MediaObject{Type: "geo"}
	case *tg.MessageMediaContact:
		return &output.MediaObject{Type: "contact"}
	default:
		return &output.MediaObject{Type: "other"}
	}
}

func documentDetail(doc *tg.Document) *output.MediaObject {
	o := &output.MediaObject{Type: "document", Size: doc.Size, MIME: doc.MimeType}
	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeFilename:
			o.FileName = a.FileName
		case *tg.DocumentAttributeSticker:
			o.Type = "sticker"
			o.Emoji = a.Alt
		case *tg.DocumentAttributeAnimated:
			o.Type = "gif"
		case *tg.DocumentAttributeVideo:
			o.Type = "video"
			if a.RoundMessage {
				o.Type = "round_video"
			}
			o.Duration = int(a.Duration)
			o.Width, o.Height = a.W, a.H
		case *tg.DocumentAttributeAudio:
			o.Type = "audio"
			if a.Voice {
				o.Type = "voice"
			}
			o.Duration = a.Duration
			o.Performer, _ = a.GetPerformer()
			o.AudioTitle, _ = a.GetTitle()
		case *tg.DocumentAttributeImageSize:
			o.Width, o.Height = a.W, a.H
		}
	}
	if o.Type == "sticker" {
		switch doc.MimeType {
		case "application/x-tgsticker":
			o.StickerType = "animated"
		case "video/webm":
			o.StickerType = "video"
		default:
			o.StickerType = "static"
		}
	}
	return o
}

func largestPhotoSize(ph *tg.Photo) (w, h int) {
	for _, s := range ph.Sizes {
		if ps, ok := s.(*tg.PhotoSize); ok {
			if ps.W*ps.H > w*h {
				w, h = ps.W, ps.H
			}
		}
	}
	return w, h
}
