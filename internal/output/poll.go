package output

import (
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// VoteResult is emitted by `tg msg vote`.
type VoteResult struct {
	Action    string `json:"action"` // "vote" | "retract"
	MessageID int    `json:"message_id"`
}

// RenderVote prints a vote/retract confirmation.
func RenderVote(io *ui.IOStreams, r VoteResult) error {
	tp := NewTablePrinter(io)
	tp.AddRow("ACTION", r.Action)
	tp.AddRow("MESSAGE_ID", strconv.Itoa(r.MessageID))
	return tp.Render()
}
