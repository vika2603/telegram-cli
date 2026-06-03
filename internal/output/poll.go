package output

import (
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// PollOption is one answer of a poll, with tallies when available.
type PollOption struct {
	Text    string `json:"text"`
	Voters  int    `json:"voters,omitempty"`
	Chosen  bool   `json:"chosen,omitempty"`  // the account voted for this option
	Correct bool   `json:"correct,omitempty"` // quiz: this is the correct answer
}

// PollInfo describes a poll: its question, options, and tallies.
type PollInfo struct {
	Question    string       `json:"question"`
	Options     []PollOption `json:"options"`
	TotalVoters int          `json:"total_voters,omitempty"`
	Multiple    bool         `json:"multiple,omitempty"`
	Quiz        bool         `json:"quiz,omitempty"`
	Public      bool         `json:"public,omitempty"`
	Closed      bool         `json:"closed,omitempty"`
	// MinResults is true when the tallies are the pre-vote "minimal" view
	// (you haven't voted; your choice / quiz answer may be withheld).
	MinResults bool `json:"min_results,omitempty"`
}

// VoteResult is emitted by `tg msg vote`.
type VoteResult struct {
	Action    string `json:"action"` // "vote" | "retract"
	MessageID int    `json:"message_id"`
}

// RenderMessageDetail prints a single message's key fields, expanding media
// detail, album members, and poll content when present.
func RenderMessageDetail(io *ui.IOStreams, r MessageRow) error {
	tp := NewTablePrinter(io)
	tp.AddRow("REF", r.Ref)
	tp.AddRow("DATE", r.Date)
	if name := messageSenderName(r); name != "" {
		tp.AddRow("FROM", name)
	}
	if r.Text != "" {
		tp.AddRow("TEXT", r.Text)
	}
	if len(r.Album) > 0 {
		tp.AddRow("ALBUM", strconv.Itoa(len(r.Album))+" items")
	} else if r.MediaDetail != nil {
		for _, kv := range mediaDetailRows(r.MediaDetail) {
			tp.AddRow(kv[0], kv[1])
		}
	}
	if err := tp.Render(); err != nil {
		return err
	}
	for i, it := range r.Album {
		_, _ = io.Out.Write([]byte("\n"))
		at := NewTablePrinter(io)
		at.AddRow("#", strconv.Itoa(i+1))
		at.AddRow("REF", it.Ref)
		if it.Media != nil {
			for _, kv := range mediaDetailRows(it.Media) {
				at.AddRow(kv[0], kv[1])
			}
		}
		if err := at.Render(); err != nil {
			return err
		}
	}
	poll := pollOf(r)
	if poll != nil {
		_, _ = io.Out.Write([]byte("\n"))
		return renderPollBody(io, *poll)
	}
	return nil
}

func pollOf(r MessageRow) *PollInfo {
	if r.MediaDetail != nil {
		return r.MediaDetail.Poll
	}
	return nil
}

// mediaDetailRows flattens a MediaObject into key/value rows for human output.
func mediaDetailRows(m *MediaObject) [][2]string {
	rows := [][2]string{{"MEDIA", m.Type}}
	add := func(k, v string) {
		if v != "" {
			rows = append(rows, [2]string{k, v})
		}
	}
	add("FILE_NAME", m.FileName)
	if m.Size > 0 {
		add("SIZE", strconv.FormatInt(m.Size, 10))
	}
	add("MIME", m.MIME)
	if m.Width > 0 || m.Height > 0 {
		add("DIMENSIONS", strconv.Itoa(m.Width)+"x"+strconv.Itoa(m.Height))
	}
	if m.Duration > 0 {
		add("DURATION", strconv.Itoa(m.Duration)+"s")
	}
	add("EMOJI", m.Emoji)
	add("STICKER_TYPE", m.StickerType)
	add("TITLE", m.Title)
	add("URL", m.URL)
	add("SITE", m.SiteName)
	return rows
}

func messageSenderName(r MessageRow) string {
	switch {
	case r.FromUsername != "":
		return "@" + r.FromUsername
	case r.FromTitle != "":
		return r.FromTitle
	case r.FromRef != "":
		return r.FromRef
	}
	return ""
}

func renderPollBody(io *ui.IOStreams, p PollInfo) error {
	tp := NewTablePrinter(io)
	tp.AddRow("QUESTION", p.Question)
	tp.AddRow("TOTAL_VOTERS", strconv.Itoa(p.TotalVoters))
	if err := tp.Render(); err != nil {
		return err
	}
	ot := NewTablePrinter(io)
	ot.AddHeader("#", "OPTION", "VOTES", "FLAGS")
	for i, o := range p.Options {
		flags := ""
		if o.Chosen {
			flags = "voted"
		}
		if o.Correct {
			if flags != "" {
				flags += ","
			}
			flags += "correct"
		}
		ot.AddRow(strconv.Itoa(i+1), o.Text, strconv.Itoa(o.Voters), flags)
	}
	return ot.Render()
}

// RenderVote prints a vote/retract confirmation.
func RenderVote(io *ui.IOStreams, r VoteResult) error {
	tp := NewTablePrinter(io)
	tp.AddRow("ACTION", r.Action)
	tp.AddRow("MESSAGE_ID", strconv.Itoa(r.MessageID))
	return tp.Render()
}
