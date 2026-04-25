package output

import (
	"encoding/json"
	"io"
)

// LogoutRow is emitted by "tg auth logout".
// DefaultCleared is true only when the purged slot was cfg.DefaultAccount and
// the pointer was successfully removed from config.toml.
type LogoutRow struct {
	Action         string `json:"action"`
	Name           string `json:"name"`
	Purged         bool   `json:"purged"`
	DefaultCleared bool   `json:"default_cleared"`
}

// WriteLogoutJSON emits a single JSON object followed by a newline.
func WriteLogoutJSON(w io.Writer, r LogoutRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}

// AuthStatusRow is emitted by "tg auth status".
// ProbeOK is only meaningful when Probed is true (omitempty keeps it absent
// from JSON when --probe was not passed). SessionModified is omitted when the
// session file does not exist (fresh NEW slot).
type AuthStatusRow struct {
	Name            string `json:"name"`
	State           string `json:"state"`
	APIID           int    `json:"api_id,omitempty"`
	Default         bool   `json:"default"`
	Probed          bool   `json:"probed"`
	ProbeOK         bool   `json:"probe_ok,omitempty"`
	SessionModified string `json:"session_modified,omitempty"`
}

// WriteAuthStatusJSON emits a single JSON object followed by a newline.
func WriteAuthStatusJSON(w io.Writer, r AuthStatusRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}
