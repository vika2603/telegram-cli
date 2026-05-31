package output

import (
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// MemberRow is the output of `chat member`. Role is one of: "creator",
// "admin", "member", "restricted", "banned", "kicked", "left".
type MemberRow struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	IsBot     bool   `json:"is_bot,omitempty"`
	Role      string `json:"role"`
	JoinedAt  string `json:"joined_at,omitempty"`
}

func RenderMembers(io *ui.IOStreams, rows []MemberRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("USER_ID", "USERNAME", "NAME", "ROLE")
	for _, r := range rows {
		tp.AddRow(
			strconv.FormatInt(r.UserID, 10),
			prefixUsername(r.Username),
			memberName(r),
			r.Role,
		)
	}
	return tp.Render()
}

func memberName(r MemberRow) string {
	switch {
	case r.FirstName != "" && r.LastName != "":
		return r.FirstName + " " + r.LastName
	case r.FirstName != "":
		return r.FirstName
	case r.LastName != "":
		return r.LastName
	}
	return ""
}
