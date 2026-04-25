package output

import (
	"github.com/vika2603/telegram-cli/internal/ui"
)

// ContactRow is emitted by `contact list` and (optionally) `contact add` as an
// echo of the newly-added contact. Matches the locked JSON field list for
// contacts discovery: id, first_name, last_name, username, phone, mutual,
// blocked, bot.
type ContactRow struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Phone     string `json:"phone,omitempty"`
	Mutual    bool   `json:"mutual,omitempty"`
	Blocked   bool   `json:"blocked,omitempty"`
	Bot       bool   `json:"bot,omitempty"`
}

func RenderContacts(io *ui.IOStreams, rows []ContactRow) error {
	tp := NewTablePrinter(io)
	tp.AddHeader("ID", "NAME", "USERNAME", "PHONE", "FLAGS")
	for _, r := range rows {
		name := r.FirstName
		if r.LastName != "" {
			if name != "" {
				name = name + " " + r.LastName
			} else {
				name = r.LastName
			}
		}
		flags := ""
		if r.Mutual {
			flags += "mutual "
		}
		if r.Blocked {
			flags += "blocked "
		}
		if r.Bot {
			flags += "bot "
		}
		tp.AddRow(i64toaOrBlank(r.ID), name, r.Username, r.Phone, flags)
	}
	return tp.Render()
}
