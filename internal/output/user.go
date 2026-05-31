// Package output defines row shapes emitted by commands, plus the paired
// TTY table renderers. Every row type is a flat
// struct with explicit `json:"..."` tags so field lists are stable and
// pinnable for CI golden tests.
package output

import (
	"fmt"
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// UserRow is the output of `tg me`, `search chat` (user rows), and the
// nested user shape inside chat member.
type UserRow struct {
	ID         int64  `json:"id"`
	Username   string `json:"username,omitempty"`
	FirstName  string `json:"first_name,omitempty"`
	LastName   string `json:"last_name,omitempty"`
	Phone      string `json:"phone,omitempty"`
	IsBot      bool   `json:"is_bot,omitempty"`
	IsSelf     bool   `json:"is_self,omitempty"`
	IsVerified bool   `json:"is_verified,omitempty"`
}

// DisplayName returns first+last, falling back to @username, falling back
// to user#id.
func (r UserRow) DisplayName() string {
	switch {
	case r.FirstName != "" && r.LastName != "":
		return r.FirstName + " " + r.LastName
	case r.FirstName != "":
		return r.FirstName
	case r.LastName != "":
		return r.LastName
	case r.Username != "":
		return "@" + r.Username
	}
	return fmt.Sprintf("user#%d", r.ID)
}

// RenderUser prints a single-user block (two columns: label, value).
func RenderUser(io *ui.IOStreams, r UserRow) error {
	tp := NewTablePrinter(io)
	tp.AddRow("ID", strconv.FormatInt(r.ID, 10))
	tp.AddRow("NAME", r.DisplayName())
	if r.Username != "" {
		tp.AddRow("USERNAME", "@"+r.Username)
	}
	if r.Phone != "" {
		tp.AddRow("PHONE", "+"+r.Phone)
	}
	switch {
	case r.IsBot:
		tp.AddRow("KIND", "bot")
	case r.IsSelf:
		tp.AddRow("KIND", "self")
	default:
		tp.AddRow("KIND", "user")
	}
	return tp.Render()
}
