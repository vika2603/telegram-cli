package output

import (
	"encoding/json"
	"io"
)

// AccountPasswordRow is emitted by password verbs. HadPrevious differentiates
// "created a new password" from "changed existing password". For disable,
// Action="password_disable" and other fields are left empty.
type AccountPasswordRow struct {
	Action           string `json:"action"` // "password_set" | "password_disable"
	HadPrevious      bool   `json:"had_previous,omitempty"`
	HasHint          bool   `json:"has_hint,omitempty"`
	HasRecoveryEmail bool   `json:"has_recovery_email,omitempty"`
}

// WriteAccountPasswordJSON encodes r as a single JSON object followed by a
// newline to w.
func WriteAccountPasswordJSON(w io.Writer, r AccountPasswordRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}
