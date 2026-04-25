package output

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// MessageRow is the output of `msg list`. Text is the message text when
// present; for media-only messages, Text is the caption (may be empty).
type MessageRow struct {
	ID           int    `json:"id"`
	Ref          string `json:"ref,omitempty"`
	Date         string `json:"date"` // RFC3339
	FromID       int64  `json:"from_id,omitempty"`
	FromRef      string `json:"from_ref,omitempty"`
	FromKind     string `json:"from_kind,omitempty"`
	FromTitle    string `json:"from_title,omitempty"`
	FromUsername string `json:"from_username,omitempty"`
	ReplyToID    int    `json:"reply_to_id,omitempty"`
	Text         string `json:"text,omitempty"`
	HasMedia     bool   `json:"has_media,omitempty"`
	MediaKind    string `json:"media_kind,omitempty"` // "photo" | "video" | "document" | "voice" | "audio" | "sticker" | "poll" | "web_page" | "other"
	Views        int    `json:"views,omitempty"`
	IsPinned     bool   `json:"is_pinned,omitempty"`
}

func (r MessageRow) MarshalJSON() ([]byte, error) {
	type messageRowJSON struct {
		Ref      string       `json:"ref,omitempty"`
		ID       int          `json:"id,omitempty"`
		Date     string       `json:"date,omitempty"`
		From     *PeerObject  `json:"from,omitempty"`
		Text     string       `json:"text,omitempty"`
		Media    *MediaObject `json:"media,omitempty"`
		ReplyTo  int          `json:"reply_to,omitempty"`
		Views    int          `json:"views,omitempty"`
		IsPinned bool         `json:"is_pinned,omitempty"`
	}
	var from *PeerObject
	if r.FromID != 0 || r.FromRef != "" || r.FromTitle != "" || r.FromUsername != "" || r.FromKind != "" {
		p := peerObject(r.FromRef, r.FromID, r.FromKind, r.FromTitle, r.FromUsername)
		from = &p
	}
	var media *MediaObject
	if r.MediaKind != "" {
		media = &MediaObject{Type: r.MediaKind}
	}
	return json.Marshal(messageRowJSON{
		Ref:      r.Ref,
		ID:       r.ID,
		Date:     r.Date,
		From:     from,
		Text:     r.Text,
		Media:    media,
		ReplyTo:  r.ReplyToID,
		Views:    r.Views,
		IsPinned: r.IsPinned,
	})
}

func MessageSummaryFromRow(r MessageRow) MessageSummary {
	s := MessageSummary{
		Ref:  r.Ref,
		ID:   r.ID,
		Date: r.Date,
		Text: r.Text,
	}
	if r.FromID != 0 || r.FromRef != "" || r.FromTitle != "" || r.FromUsername != "" || r.FromKind != "" {
		p := peerObject(r.FromRef, r.FromID, r.FromKind, r.FromTitle, r.FromUsername)
		s.From = &p
	}
	if r.MediaKind != "" {
		s.Media = &MediaObject{Type: r.MediaKind}
	}
	return s
}

func RenderMessages(io *ui.IOStreams, rows []MessageRow) error {
	if io.IsStdoutTTY() {
		return renderMessagesTTY(io, rows)
	}
	tp := NewTablePrinter(io)
	tp.AddHeader("REF", "DATE", "FROM", "TEXT")
	for _, r := range rows {
		tp.AddRow(
			displayMessageRef(r),
			r.Date,
			shortText(displayMessageFrom(r), 36),
			shortText(messagePreview(r), 60),
		)
	}
	return tp.Render()
}

func renderMessagesTTY(io *ui.IOStreams, rows []MessageRow) error {
	width := io.TerminalWidth()
	if width <= 0 {
		width = 80
	}
	colors := io.ColorScheme()
	for i, r := range rows {
		if i > 0 {
			fmt.Fprintln(io.Out)
		}
		ref := displayMessageRef(r)
		header := messageHeader(r)
		if header == "" {
			fmt.Fprintln(io.Out, colors.Bold(ref))
		} else if displayWidth(ref)+displayWidth(header)+2 <= width {
			fmt.Fprintf(io.Out, "%s  %s\n", colors.Bold(ref), colors.Gray(fitText(header, width-displayWidth(ref)-2)))
		} else {
			fmt.Fprintln(io.Out, colors.Bold(ref))
			fmt.Fprintf(io.Out, "  %s\n", colors.Gray(fitText(header, width-2)))
		}
		body := messagePreview(r)
		if body == "" {
			body = "[empty]"
		}
		fmt.Fprintf(io.Out, "  %s\n", fitText(oneLine(body), max(width-2, 16)))
	}
	return nil
}

func RenderDigest(io *ui.IOStreams, rows []MessageRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("DATE", "FROM", "MESSAGE")
	for _, r := range rows {
		tp.AddRow(
			r.Date,
			shortText(displayMessageFrom(r), 32),
			shortText(messagePreview(r), 96),
		)
	}
	return tp.Render()
}

func displayMessageRef(r MessageRow) string {
	if r.Ref != "" {
		return r.Ref
	}
	return strconv.Itoa(r.ID)
}

func displayMessageFrom(r MessageRow) string {
	username := prefixUsername(r.FromUsername)
	switch {
	case r.FromTitle != "" && username != "":
		return r.FromTitle + " (" + username + ")"
	case r.FromTitle != "":
		return r.FromTitle
	case username != "":
		return username
	case r.FromRef != "":
		return r.FromRef
	case r.FromID != 0:
		return strconv.FormatInt(r.FromID, 10)
	default:
		return ""
	}
}

func messagePreview(r MessageRow) string {
	if !r.HasMedia || r.MediaKind == "" {
		return r.Text
	}
	media := "[" + r.MediaKind + "]"
	if r.Text == "" {
		return media
	}
	return media + " " + r.Text
}

func messageHeader(r MessageRow) string {
	parts := make([]string, 0, 5)
	if from := displayMessageFrom(r); from != "" {
		parts = append(parts, from)
	}
	if when := shortMessageDate(r.Date); when != "" {
		parts = append(parts, when)
	}
	if r.ReplyToID > 0 {
		parts = append(parts, "reply "+strconv.Itoa(r.ReplyToID))
	}
	if r.Views > 0 {
		parts = append(parts, strconv.Itoa(r.Views)+" views")
	}
	if r.IsPinned {
		parts = append(parts, "pinned")
	}
	return strings.Join(parts, " · ")
}

func shortMessageDate(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Local().Format("01-02 15:04")
}

// shortText collapses whitespace and truncates by terminal display width so
// wide CJK and emoji text does not overflow compact tables as aggressively.
func shortText(s string, n int) string {
	return fitText(oneLine(s), n)
}
