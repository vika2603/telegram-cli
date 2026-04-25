package output

import (
	"encoding/json"
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// SearchMsgRow is the output of `search msg`. It embeds MessageRow and
// adds chat identification because search results span chats.
type SearchMsgRow struct {
	MessageID int    `json:"message_id"`
	ChatID    int64  `json:"chat_id"`
	ChatTitle string `json:"chat_title,omitempty"`
	ChatKind  string `json:"chat_kind,omitempty"`
	Date      string `json:"date"`
	FromID    int64  `json:"from_id,omitempty"`
	Text      string `json:"text,omitempty"`
	MediaKind string `json:"media_kind,omitempty"`
}

func RenderSearchMsg(io *ui.IOStreams, rows []SearchMsgRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("ID", "CHAT", "DATE", "TEXT")
	for _, r := range rows {
		chat := r.ChatTitle
		if chat == "" {
			chat = strconv.FormatInt(r.ChatID, 10)
		}
		tp.AddRow(
			strconv.Itoa(r.MessageID),
			chat,
			r.Date,
			shortText(r.Text, 60),
		)
	}
	return tp.Render()
}

// SearchChatRow is the output of `search chat`. Source is "my" (came
// from ContactsFound.MyResults) or "public" (came from .Results).
type SearchChatRow struct {
	ChatRow
	Source string `json:"source"`
}

func (r SearchChatRow) MarshalJSON() ([]byte, error) {
	type searchChatRowJSON struct {
		Peer   PeerObject `json:"peer"`
		Title  string     `json:"title,omitempty"`
		Type   string     `json:"type,omitempty"`
		Source string     `json:"source"`
	}
	return json.Marshal(searchChatRowJSON{
		Peer:   peerObject(r.Ref, r.ID, r.Kind, r.Title, r.Username),
		Title:  r.Title,
		Type:   r.Kind,
		Source: r.Source,
	})
}

func RenderSearchChat(io *ui.IOStreams, rows []SearchChatRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("ID", "KIND", "TITLE", "USERNAME", "SOURCE")
	for _, r := range rows {
		tp.AddRow(
			strconv.FormatInt(r.ID, 10),
			r.Kind,
			r.Title,
			prefixUsername(r.Username),
			r.Source,
		)
	}
	return tp.Render()
}
