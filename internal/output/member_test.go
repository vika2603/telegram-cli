package output_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestRenderMembers_EmitsHeaderAndRows(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.MemberRow{
		{UserID: 1, Username: "alice", FirstName: "Alice", Role: "admin"},
		{UserID: 2, Username: "bob", FirstName: "Bob", Role: "member"},
	}
	require.NoError(t, output.RenderMembers(ios, rows))
	got := stdout.String()
	require.Contains(t, got, "USER_ID")
	require.Contains(t, got, "Alice")
	require.Contains(t, got, "admin")
}
