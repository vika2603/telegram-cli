package output

import (
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// StickerRow is one row of `tg msg sticker list`. For Kind=="sticker" the Ref
// is a self-contained handle to pass to `msg send --sticker`; for Kind=="set"
// the row describes an installed sticker set (browse only).
type StickerRow struct {
	Kind  string `json:"kind"`            // "sticker" | "set"
	Ref   string `json:"ref,omitempty"`   // sendable handle (sticker)
	ID    int64  `json:"id,omitempty"`    // document id (sticker)
	Emoji string `json:"emoji,omitempty"` // associated emoji (sticker)
	Type  string `json:"type,omitempty"`  // "static" | "animated" | "video" (sticker)
	Set   string `json:"set,omitempty"`   // set short name (set)
	Title string `json:"title,omitempty"` // set title (set)
	Count int    `json:"count,omitempty"` // stickers in set (set)
}

// FaveResult is emitted by `tg msg sticker fave` / `unfave`.
type FaveResult struct {
	Action string `json:"action"` // "fave" | "unfave"
	ID     int64  `json:"id"`
}

// RenderFave prints a fave/unfave confirmation.
func RenderFave(io *ui.IOStreams, r FaveResult) error {
	tp := NewTablePrinter(io)
	tp.AddRow("ACTION", r.Action)
	tp.AddRow("ID", strconv.FormatInt(r.ID, 10))
	return tp.Render()
}

// RenderStickers prints sticker rows (or set rows) as a table.
func RenderStickers(io *ui.IOStreams, rows []StickerRow) error {
	tp := NewTablePrinter(io)
	sets := len(rows) > 0 && rows[0].Kind == "set"
	if sets {
		tp.AddHeader("SET", "TITLE", "COUNT")
		for _, r := range rows {
			tp.AddRow(r.Set, r.Title, strconv.Itoa(r.Count))
		}
	} else {
		tp.AddHeader("EMOJI", "TYPE", "REF")
		for _, r := range rows {
			tp.AddRow(r.Emoji, r.Type, r.Ref)
		}
	}
	return tp.Render()
}
