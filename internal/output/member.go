package output

import (
	"strconv"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// MemberRow is the output of `chat member list`. Role is one of: "creator",
// "admin", "member", "restricted", "banned", "kicked", "left".
type MemberRow struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	IsBot     bool   `json:"is_bot,omitempty"`
	Role      string `json:"role"`
	Rank      string `json:"rank,omitempty"` // custom admin title, if any
	JoinedAt  string `json:"joined_at,omitempty"`
}

func RenderMembers(io *ui.IOStreams, rows []MemberRow) error {
	showRank := false
	for _, r := range rows {
		if r.Rank != "" {
			showRank = true
			break
		}
	}
	tp := NewTablePrinter(io)
	if showRank {
		tp.AddHeader("USER_ID", "USERNAME", "NAME", "ROLE", "RANK")
	} else {
		tp.AddHeader("USER_ID", "USERNAME", "NAME", "ROLE")
	}
	for _, r := range rows {
		cells := []string{
			strconv.FormatInt(r.UserID, 10),
			prefixUsername(r.Username),
			memberName(r),
			r.Role,
		}
		if showRank {
			cells = append(cells, r.Rank)
		}
		tp.AddRow(cells...)
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
