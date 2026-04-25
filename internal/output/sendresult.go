package output

import (
	"encoding/json"
	"io"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// SendResultRow is the shared JSON / TTY row emitted by write-side verbs
// that create or modify a single message (send / edit / forward / react / upload).
type SendResultRow struct {
	Action    string `json:"action"` // "send" | "edit" | "forward" | "react" | "upload"
	MessageID int    `json:"message_id,omitempty"`
	ChatID    int64  `json:"chat_id,omitempty"`
	Date      string `json:"date,omitempty"` // RFC3339
}

// WriteSendResultJSON encodes r as a single JSON object to w.
func WriteSendResultJSON(w io.Writer, r SendResultRow) error {
	return json.NewEncoder(w).Encode(r)
}

// RenderSendResults prints rows as an aligned TTY table (or tab-separated in
// non-TTY mode) using the shared TablePrinter.
func RenderSendResults(io *ui.IOStreams, rows []SendResultRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("ACTION", "MESSAGE_ID", "CHAT_ID", "DATE")
	for _, r := range rows {
		tp.AddRow(r.Action, itoaOrBlank(r.MessageID), i64toaOrBlank(r.ChatID), r.Date)
	}
	return tp.Render()
}
