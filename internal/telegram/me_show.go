package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
)

// ShowMe loads the current Telegram identity.
func ShowMe(ctx context.Context, api *tg.Client) (output.UserRow, error) {
	full, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
	if err != nil {
		return output.UserRow{}, err
	}
	u, ok := full.Users[0].(*tg.User)
	if !ok {
		return output.UserRow{}, fmt.Errorf("%w: unexpected self type", command.ErrPrecondition)
	}
	return UserRow(u, true), nil
}

// UserRow converts a Telegram user to the public output row shape.
func UserRow(u *tg.User, isSelf bool) output.UserRow {
	return output.UserRow{
		ID:         u.ID,
		Username:   u.Username,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		Phone:      u.Phone,
		IsBot:      u.Bot,
		IsSelf:     isSelf,
		IsVerified: u.Verified,
	}
}
