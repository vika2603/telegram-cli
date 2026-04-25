package telegram

import (
	"testing"

	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/output"
)

func TestUserToContactRow(t *testing.T) {
	row := userToContactRow(&tg.User{
		ID:        10,
		FirstName: "Alice",
		LastName:  "Smith",
		Username:  "alice",
		Phone:     "+15551234",
		Bot:       true,
	}, true)
	require.Equal(t, output.ContactRow{
		ID:        10,
		FirstName: "Alice",
		LastName:  "Smith",
		Username:  "alice",
		Phone:     "+15551234",
		Mutual:    true,
		Bot:       true,
	}, row)
}

func TestFillContactRowFromUserPreservesBlockedFlag(t *testing.T) {
	row := output.ContactRow{Blocked: true}
	fillContactRowFromUser(&row, &tg.User{ID: 11, FirstName: "Bob"})
	require.Equal(t, output.ContactRow{ID: 11, FirstName: "Bob", Blocked: true}, row)
}
