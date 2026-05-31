package output

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// TopicRow is the output of `tg chat topic list` and
// `tg chat topic create`. ID is the forum topic id, which equals the id of
// the topic's creation service message.
type TopicRow struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	IconColor   int    `json:"icon_color,omitempty"`
	IconEmojiID int64  `json:"icon_emoji_id,omitempty"`
	TopMessage  int    `json:"top_message,omitempty"`
	UnreadCount int    `json:"unread_count,omitempty"`
	Closed      bool   `json:"closed,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
	Pinned      bool   `json:"pinned,omitempty"`
}

// RenderTopicList prints forum topics as a table.
func RenderTopicList(io *ui.IOStreams, rows []TopicRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("ID", "TITLE", "TOP_MSG", "UNREAD", "FLAGS")
	for _, r := range rows {
		tp.AddRow(
			strconv.Itoa(r.ID),
			r.Title,
			itoaOrBlank(r.TopMessage),
			itoaOrBlank(r.UnreadCount),
			topicFlags(r),
		)
	}
	return tp.Render()
}

// RenderTopic prints a single forum topic (used by `topics create`).
func RenderTopic(io *ui.IOStreams, r TopicRow) error {
	tp := NewTablePrinter(io)
	tp.AddRow("ID", strconv.Itoa(r.ID))
	tp.AddRow("TITLE", r.Title)
	if r.IconColor != 0 {
		tp.AddRow("ICON_COLOR", fmt.Sprintf("#%06X", r.IconColor))
	}
	return tp.Render()
}

func topicFlags(r TopicRow) string {
	var f []string
	if r.Pinned {
		f = append(f, "pinned")
	}
	if r.Closed {
		f = append(f, "closed")
	}
	if r.Hidden {
		f = append(f, "hidden")
	}
	return strings.Join(f, ",")
}
