package output_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestSendResultRow_RenderTTY(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.SendResultRow{
		{Action: "send", MessageID: 4242, ChatID: -100123, Date: "2026-04-24T10:00:00Z"},
	}
	require.NoError(t, output.RenderSendResults(ios, rows))
	out := stdout.String()
	require.Contains(t, out, "4242")
	require.Contains(t, out, "send")
}

func TestSendResultRow_Fields(t *testing.T) {
	r := output.SendResultRow{Action: "edit", MessageID: 7, ChatID: 5}
	var buf bytes.Buffer
	require.NoError(t, output.WriteSendResultJSON(&buf, r))
	s := buf.String()
	require.Contains(t, s, `"action":"edit"`)
	require.Contains(t, s, `"message_id":7`)
	require.Contains(t, s, `"chat_id":5`)
	require.NotContains(t, s, `"date"`)
}
