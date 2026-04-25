package output

import (
	"encoding/json"
	"io"
)

// AccountSessionRow is one row from `session list`. Hash is stringified
// (Telegram uses int64; shell users may lose precision if it's emitted as
// JSON number).
type AccountSessionRow struct {
	Hash            string `json:"hash"`
	Current         bool   `json:"current"`
	OfficialApp     bool   `json:"official_app,omitempty"`
	PasswordPending bool   `json:"password_pending,omitempty"`
	DeviceModel     string `json:"device_model"`
	Platform        string `json:"platform"`
	SystemVersion   string `json:"system_version,omitempty"`
	AppName         string `json:"app_name"`
	AppVersion      string `json:"app_version,omitempty"`
	APIID           int    `json:"api_id,omitempty"`
	Country         string `json:"country,omitempty"`
	Region          string `json:"region,omitempty"`
	IP              string `json:"ip,omitempty"`
	DateCreated     string `json:"date_created"` // RFC3339
	DateActive      string `json:"date_active"`  // RFC3339
	// Unconfirmed distinguishes sessions awaiting approval from active ones.
	// Telegram surfaces it separately from Current; scripts that want "only
	// usable sessions" filter on `unconfirmed == false` (or absent).
	Unconfirmed bool `json:"unconfirmed,omitempty"`
}

// WriteAccountSessionJSON emits one ndjson line.
func WriteAccountSessionJSON(w io.Writer, r AccountSessionRow) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}
	return nil
}
