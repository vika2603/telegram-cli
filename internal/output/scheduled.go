package output

import (
	"github.com/vika2603/telegram-cli/internal/ui"
)

// ScheduledMessageRow mirrors MessageRow but exposes the scheduled-for time
// alongside the stored date. Emitted by `msg schedule-list`.
type ScheduledMessageRow struct {
	ID           int    `json:"id"`
	Date         string `json:"date"`
	ScheduledFor string `json:"scheduled_for"`
	Text         string `json:"text,omitempty"`
	FromID       int64  `json:"from_id,omitempty"`
}

func RenderScheduled(io *ui.IOStreams, rows []ScheduledMessageRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("ID", "SCHEDULED_FOR", "TEXT")
	for _, r := range rows {
		tp.AddRow(itoaOrBlank(r.ID), r.ScheduledFor, shortText(r.Text, 60))
	}
	return tp.Render()
}
