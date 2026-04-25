package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
)

func TestProfileRow_Fields(t *testing.T) {
	r := output.ProfileRow{
		Action:    "set-name",
		FirstName: "Bob",
		LastName:  "Jones",
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteProfileJSON(&buf, r))
	s := buf.String()
	require.Contains(t, s, `"action":"set-name"`)
	require.Contains(t, s, `"first_name":"Bob"`)
	require.Contains(t, s, `"last_name":"Jones"`)
}

func TestProfileRow_OmitemptyPin(t *testing.T) {
	// Only Action populated — every other field must be omitted from JSON.
	r := output.ProfileRow{Action: "delete-photo"}
	b, err := json.Marshal(r)
	require.NoError(t, err)
	s := string(b)
	require.Contains(t, s, `"action":"delete-photo"`)
	require.NotContains(t, s, `"first_name"`)
	require.NotContains(t, s, `"last_name"`)
	require.NotContains(t, s, `"username"`)
	require.NotContains(t, s, `"bio"`)
	require.NotContains(t, s, `"status"`)
	require.NotContains(t, s, `"photo_id"`)
}

func TestProfileRow_PhotoIDPopulated(t *testing.T) {
	r := output.ProfileRow{Action: "set-photo", PhotoID: 9001}
	b, err := json.Marshal(r)
	require.NoError(t, err)
	s := string(b)
	require.Contains(t, s, `"action":"set-photo"`)
	require.Contains(t, s, `"photo_id":9001`)
}
