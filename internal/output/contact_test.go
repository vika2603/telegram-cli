package output_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestRenderContacts_TTY(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.ContactRow{
		{ID: 10, FirstName: "Alice", LastName: "Smith", Username: "alice", Phone: "+15551234", Mutual: true},
		{ID: 11, FirstName: "Bob", Username: "", Phone: "+15551235", Blocked: true},
	}
	require.NoError(t, output.RenderContacts(ios, rows))
	s := stdout.String()
	require.Contains(t, s, "Alice")
	require.Contains(t, s, "Bob")
}

func TestRenderContacts_LastNameOnly(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.ContactRow{
		{ID: 99, LastName: "Smith"},
	}
	require.NoError(t, output.RenderContacts(ios, rows))
	s := stdout.String()
	require.Contains(t, s, "Smith")
	// ensure no leading-space artifact: " Smith" must not appear between the
	// USERNAME column start and the actual name — we approximate by asserting
	// the exact 2-char prefix " S" (space + S) does not appear anywhere.
	require.NotContains(t, s, " Smith")
}

func TestRenderContacts_BotFlag(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	rows := []output.ContactRow{
		{ID: 200, FirstName: "Helper", Bot: true},
	}
	require.NoError(t, output.RenderContacts(ios, rows))
	s := stdout.String()
	require.Contains(t, s, "Helper")
	require.Contains(t, s, "bot")
}

func TestContactRow_JSON(t *testing.T) {
	r := output.ContactRow{ID: 42, FirstName: "Carol", Mutual: true}
	b, err := json.Marshal(r)
	require.NoError(t, err)
	s := string(b)
	require.Contains(t, s, `"id":42`)
	require.Contains(t, s, `"first_name":"Carol"`)
	require.Contains(t, s, `"mutual":true`)
	require.NotContains(t, s, `"last_name"`)
	require.NotContains(t, s, `"username"`)
	require.NotContains(t, s, `"phone"`)
	require.NotContains(t, s, `"blocked"`)
	require.NotContains(t, s, `"bot"`)
}
