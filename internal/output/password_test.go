package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
)

func TestWriteAccountPasswordJSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		row  output.AccountPasswordRow
	}{
		{
			name: "password_set_new",
			row: output.AccountPasswordRow{
				Action:           "password_set",
				HadPrevious:      false,
				HasHint:          true,
				HasRecoveryEmail: false,
			},
		},
		{
			name: "password_set_change",
			row: output.AccountPasswordRow{
				Action:           "password_set",
				HadPrevious:      true,
				HasHint:          true,
				HasRecoveryEmail: true,
			},
		},
		{
			name: "password_disable",
			row: output.AccountPasswordRow{
				Action: "password_disable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := output.WriteAccountPasswordJSON(&buf, tt.row)
			require.NoError(t, err)

			// Output must end with newline.
			out := buf.String()
			require.True(t, len(out) > 0 && out[len(out)-1] == '\n', "output must end with newline")

			// Must round-trip via JSON.
			var got output.AccountPasswordRow
			err = json.Unmarshal(buf.Bytes(), &got)
			require.NoError(t, err)
			require.Equal(t, tt.row.Action, got.Action)
			require.Equal(t, tt.row.HadPrevious, got.HadPrevious)
			require.Equal(t, tt.row.HasHint, got.HasHint)
			require.Equal(t, tt.row.HasRecoveryEmail, got.HasRecoveryEmail)
		})
	}
}

func TestWriteAccountPasswordJSON_OmitsZeroOptionals(t *testing.T) {
	row := output.AccountPasswordRow{Action: "password_set"}
	var buf bytes.Buffer
	require.NoError(t, output.WriteAccountPasswordJSON(&buf, row))

	// omitempty: had_previous, has_hint, has_recovery_email must not appear
	// when false.
	m := map[string]any{}
	require.NoError(t, json.Unmarshal(buf.Bytes(), &m))
	_, hasPrev := m["had_previous"]
	_, hasHint := m["has_hint"]
	_, hasRecovery := m["has_recovery_email"]
	require.False(t, hasPrev, "had_previous should be omitted when false")
	require.False(t, hasHint, "has_hint should be omitted when false")
	require.False(t, hasRecovery, "has_recovery_email should be omitted when false")
}
