package output

import (
	"encoding/json"
	"io"
)

// JoinResultRow is emitted by `chat join approve` / `deny`, one per decided
// user. All is true for a bulk (--all) decision; otherwise Peer is the user.
// Error is set (and the command still exits 0) when a specific user's
// decision failed, so the rest of a multi-user batch is still reported.
type JoinResultRow struct {
	Action string   `json:"action"` // "approve" | "deny"
	All    bool     `json:"all,omitempty"`
	Peer   *PeerRef `json:"peer,omitempty"`
	Error  string   `json:"error,omitempty"`
}

// WriteJoinResultJSON emits one ndjson line.
func WriteJoinResultJSON(w io.Writer, r JoinResultRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}
