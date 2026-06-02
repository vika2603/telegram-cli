package output

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/vika2603/telegram-cli/internal/ui"
)

// RightsRow is emitted by `chat restrict` / `unrestrict` / `perms`. Peer is set
// for per-user restrict/unrestrict (absent for group-default perms). Denied
// lists the permission keywords currently revoked; Until is the restriction
// expiry (empty = permanent).
type RightsRow struct {
	Action string   `json:"action"` // "set-perms" | "unset-perms" | "perms"
	Peer   *PeerRef `json:"peer,omitempty"`
	Denied []string `json:"denied,omitempty"`
	Until  string   `json:"until,omitempty"`
}

// WriteRightsJSON emits one ndjson line.
func WriteRightsJSON(w io.Writer, r RightsRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

// RenderRights prints a single rights result as key/value rows.
func RenderRights(io *ui.IOStreams, r RightsRow) error {
	tp := NewTablePrinter(io)
	tp.AddRow("ACTION", r.Action)
	if r.Peer != nil {
		name := r.Peer.Ref
		if r.Peer.Username != "" {
			name = "@" + r.Peer.Username
		}
		tp.AddRow("PEER", name)
	}
	denied := "(none)"
	if len(r.Denied) > 0 {
		denied = strings.Join(r.Denied, ", ")
	}
	tp.AddRow("DENIED", denied)
	if r.Until != "" {
		tp.AddRow("UNTIL", r.Until)
	}
	return tp.Render()
}
