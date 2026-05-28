package output

import (
	"encoding/json"
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// SearchMsgRow is the output of `tg search msg`. The fields mirror
// MessageRow (so PR #2's forward / entities / buttons surface here too)
// and add ChatID / ChatTitle / ChatKind because search results span
// chats. We do not embed MessageRow directly because MessageRow uses
// `id` for the message id and the existing `tg search msg` shape uses
// `message_id`; embedding would either collide on the tag or force a
// breaking rename. The fields are populated by reusing messageToRow
// internally — see searchMessageElemToRow.
type SearchMsgRow struct {
	// Message identification
	MessageID int    `json:"message_id"`
	Date      string `json:"date"`
	EditDate  string `json:"edit_date,omitempty"`
	GroupedID int64  `json:"grouped_id,omitempty"`

	// Sender
	FromID       int64  `json:"from_id,omitempty"`
	FromRef      string `json:"from_ref,omitempty"`
	FromKind     string `json:"from_kind,omitempty"`
	FromTitle    string `json:"from_title,omitempty"`
	FromUsername string `json:"from_username,omitempty"`

	// Chat (added because search spans chats)
	ChatID    int64  `json:"chat_id"`
	ChatTitle string `json:"chat_title,omitempty"`
	ChatKind  string `json:"chat_kind,omitempty"`

	// Content
	Text      string          `json:"text,omitempty"`
	ReplyToID int             `json:"reply_to_id,omitempty"`
	Entities  []MessageEntity `json:"entities,omitempty"`
	Buttons   []MessageButton `json:"buttons,omitempty"`
	Forward   *ForwardInfo    `json:"forward,omitempty"`
	Reactions []ReactionCount `json:"reactions,omitempty"`

	// Media
	HasMedia  bool   `json:"has_media,omitempty"`
	MediaKind string `json:"media_kind,omitempty"`

	// Metadata
	Views    int  `json:"views,omitempty"`
	IsPinned bool `json:"is_pinned,omitempty"`
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
