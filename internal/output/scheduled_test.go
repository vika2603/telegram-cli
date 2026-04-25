package output_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestRenderScheduled_TTY(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.ScheduledMessageRow{
		{ID: 8000, Date: "2026-05-01T00:00:00Z", ScheduledFor: "2026-05-02T09:00:00Z", Text: "pay rent"},
	}
	require.NoError(t, output.RenderScheduled(ios, rows))
	s := stdout.String()
	require.Contains(t, s, "8000")
	require.Contains(t, s, "pay rent")
	require.Contains(t, s, "2026-05-02T09:00:00Z")
}

func TestScheduledMessageRow_Fields(t *testing.T) {
	// populated row keeps all core fields visible.
	r := output.ScheduledMessageRow{
		ID:           42,
		Date:         "2026-05-01T00:00:00Z",
		ScheduledFor: "2026-05-02T09:00:00Z",
		Text:         "hi",
		FromID:       7,
	}
	b, err := json.Marshal(r)
	require.NoError(t, err)
	s := string(b)
	require.Contains(t, s, `"id":42`)
	require.Contains(t, s, `"date":"2026-05-01T00:00:00Z"`)
	require.Contains(t, s, `"scheduled_for":"2026-05-02T09:00:00Z"`)
	require.Contains(t, s, `"text":"hi"`)
	require.Contains(t, s, `"from_id":7`)
}

func TestScheduledMessageRow_OmitemptyPin(t *testing.T) {
	// Only ID and the two dates populated; Text and FromID omitted.
	r := output.ScheduledMessageRow{
		ID:           1,
		Date:         "2026-05-01T00:00:00Z",
		ScheduledFor: "2026-05-02T09:00:00Z",
	}
	b, err := json.Marshal(r)
	require.NoError(t, err)
	s := string(b)
	require.Contains(t, s, `"id":1`)
	require.Contains(t, s, `"date"`)
	require.Contains(t, s, `"scheduled_for"`)
	require.NotContains(t, s, `"text"`)
	require.NotContains(t, s, `"from_id"`)
}
