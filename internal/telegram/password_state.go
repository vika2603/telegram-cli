package telegram

import (
	"context"

	"github.com/gotd/td/tg"
)

// PasswordEnabled reports whether the account currently has a 2FA password.
func PasswordEnabled(ctx context.Context, api *tg.Client) (bool, error) {
	p, err := api.AccountGetPassword(ctx)
	if err != nil {
		return false, err
	}
	return p.HasPassword, nil
}
