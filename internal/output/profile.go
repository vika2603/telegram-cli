package output

import (
	"encoding/json"
	"io"
)

// ProfileRow is the scalar result from profile-write verbs (set-name,
// set-username, set-bio, set-photo, delete-photo, set-status). Only the
// fields relevant to the action are populated; the rest omit via omitempty.
type ProfileRow struct {
	Action    string `json:"action"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
	Bio       string `json:"bio,omitempty"`
	Status    string `json:"status,omitempty"` // "online" | "offline"
	PhotoID   int64  `json:"photo_id,omitempty"`
}

// WriteProfileJSON encodes r as a single JSON object to w.
func WriteProfileJSON(w io.Writer, r ProfileRow) error {
	return json.NewEncoder(w).Encode(r)
}
