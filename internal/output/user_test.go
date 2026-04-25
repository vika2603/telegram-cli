package output_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestUserRow_Render_IncludesFields(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	row := output.UserRow{
		ID:         42,
		Username:   "durov",
		FirstName:  "Pavel",
		LastName:   "Durov",
		Phone:      "",
		IsBot:      false,
		IsSelf:     false,
		IsVerified: false,
	}
	require.NoError(t, output.RenderUser(ios, row))
	got := stdout.String()
	require.Contains(t, got, "Pavel Durov")
	require.Contains(t, got, "@durov")
}
