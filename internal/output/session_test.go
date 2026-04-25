package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
)

func TestWriteAccountSessionJSON_CurrentRow(t *testing.T) {
	row := output.AccountSessionRow{
		Hash:        "1234567890",
		Current:     true,
		DeviceModel: "Desktop",
		Platform:    "Linux",
		AppName:     "Telegram",
		DateCreated: "2024-01-01T00:00:00Z",
		DateActive:  "2024-06-01T12:00:00Z",
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteAccountSessionJSON(&buf, row))

	var got output.AccountSessionRow
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))

	require.Equal(t, "1234567890", got.Hash)
	require.True(t, got.Current)
	require.Equal(t, "Desktop", got.DeviceModel)
	require.Equal(t, "Linux", got.Platform)
	require.Equal(t, "Telegram", got.AppName)
	require.Equal(t, "2024-01-01T00:00:00Z", got.DateCreated)
	require.Equal(t, "2024-06-01T12:00:00Z", got.DateActive)
	// omitempty fields absent when zero
	require.Empty(t, got.SystemVersion)
	require.Empty(t, got.AppVersion)
	require.Empty(t, got.Country)
	require.Empty(t, got.Region)
	require.Empty(t, got.IP)
	require.Zero(t, got.APIID)
}

func TestWriteAccountSessionJSON_NonCurrentRow(t *testing.T) {
	row := output.AccountSessionRow{
		Hash:            "9876543210",
		Current:         false,
		OfficialApp:     true,
		PasswordPending: false,
		DeviceModel:     "iPhone",
		Platform:        "iOS",
		SystemVersion:   "17.0",
		AppName:         "Telegram",
		AppVersion:      "10.3.2",
		APIID:           2496,
		Country:         "US",
		Region:          "CA",
		IP:              "1.2.3.4",
		DateCreated:     "2023-05-15T08:30:00Z",
		DateActive:      "2024-03-20T14:00:00Z",
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteAccountSessionJSON(&buf, row))

	var got output.AccountSessionRow
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))

	require.Equal(t, "9876543210", got.Hash)
	require.False(t, got.Current)
	require.True(t, got.OfficialApp)
	require.Equal(t, "iPhone", got.DeviceModel)
	require.Equal(t, "iOS", got.Platform)
	require.Equal(t, "17.0", got.SystemVersion)
	require.Equal(t, "Telegram", got.AppName)
	require.Equal(t, "10.3.2", got.AppVersion)
	require.Equal(t, 2496, got.APIID)
	require.Equal(t, "US", got.Country)
	require.Equal(t, "CA", got.Region)
	require.Equal(t, "1.2.3.4", got.IP)
	require.Equal(t, "2023-05-15T08:30:00Z", got.DateCreated)
	require.Equal(t, "2024-03-20T14:00:00Z", got.DateActive)
}

func TestWriteAccountSessionJSON_NewlineTerminated(t *testing.T) {
	row := output.AccountSessionRow{
		Hash:        "42",
		DeviceModel: "Mac",
		Platform:    "macOS",
		AppName:     "Telegram",
		DateCreated: "2024-01-01T00:00:00Z",
		DateActive:  "2024-01-01T00:00:00Z",
	}
	var buf bytes.Buffer
	require.NoError(t, output.WriteAccountSessionJSON(&buf, row))
	require.True(t, bytes.HasSuffix(buf.Bytes(), []byte("\n")), "output must end with newline")
}
