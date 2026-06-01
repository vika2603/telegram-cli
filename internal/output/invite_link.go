package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// InviteLinkRow is emitted by `chat invite create/list/revoke/delete`.
// Action is set for single-result commands ("create"/"revoke"/"delete") and
// empty for list rows.
type InviteLinkRow struct {
	Action        string `json:"action,omitempty"`
	Link          string `json:"link"`
	Title         string `json:"title,omitempty"`
	Revoked       bool   `json:"revoked,omitempty"`
	Permanent     bool   `json:"permanent,omitempty"`
	RequestNeeded bool   `json:"request_needed,omitempty"`
	ExpireDate    string `json:"expire_date,omitempty"`
	UsageLimit    int    `json:"usage_limit,omitempty"`
	Usage         int    `json:"usage,omitempty"`
	Requested     int    `json:"requested,omitempty"`
	AdminID       int64  `json:"admin_id,omitempty"`
	CreatedAt     string `json:"created_at,omitempty"`
}

// WriteInviteLinkJSON emits one ndjson line.
func WriteInviteLinkJSON(w io.Writer, r InviteLinkRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

// RenderInviteLinkList prints invite links as a table.
func RenderInviteLinkList(io *ui.IOStreams, rows []InviteLinkRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("LINK", "TITLE", "USAGE", "STATE")
	for _, r := range rows {
		tp.AddRow(r.Link, r.Title, inviteUsage(r), inviteState(r))
	}
	return tp.Render()
}

// RenderInviteLink prints a single invite link as key/value rows.
func RenderInviteLink(io *ui.IOStreams, r InviteLinkRow) error {
	tp := NewTablePrinter(io)
	if r.Action != "" {
		tp.AddRow("ACTION", r.Action)
	}
	tp.AddRow("LINK", r.Link)
	if r.Title != "" {
		tp.AddRow("TITLE", r.Title)
	}
	if r.ExpireDate != "" {
		tp.AddRow("EXPIRES", r.ExpireDate)
	}
	if r.UsageLimit > 0 {
		tp.AddRow("USAGE_LIMIT", strconv.Itoa(r.UsageLimit))
	}
	if r.Usage > 0 {
		tp.AddRow("USAGE", strconv.Itoa(r.Usage))
	}
	if r.Requested > 0 {
		tp.AddRow("REQUESTED", strconv.Itoa(r.Requested))
	}
	if r.RequestNeeded {
		tp.AddRow("REQUEST_NEEDED", "true")
	}
	if r.Revoked {
		tp.AddRow("REVOKED", "true")
	}
	return tp.Render()
}

func inviteUsage(r InviteLinkRow) string {
	if r.UsageLimit > 0 {
		return fmt.Sprintf("%d/%d", r.Usage, r.UsageLimit)
	}
	if r.Usage > 0 {
		return strconv.Itoa(r.Usage)
	}
	return ""
}

func inviteState(r InviteLinkRow) string {
	switch {
	case r.Revoked:
		return "revoked"
	case r.Permanent:
		return "permanent"
	case r.RequestNeeded:
		return "request"
	default:
		return "active"
	}
}
