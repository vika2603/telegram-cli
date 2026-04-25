package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"

	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
)

// SetPassword performs Telegram's SRP ceremony and applies a new 2FA password.
func SetPassword(ctx context.Context, api *tg.Client, current, next, hint string) (bool, error) {
	p, err := api.AccountGetPassword(ctx)
	if err != nil {
		return false, err
	}
	algo, ok := p.NewAlgo.(*tg.PasswordKdfAlgoSHA256SHA256PBKDF2HMACSHA512iter100000SHA256ModPow)
	if !ok {
		return false, fmt.Errorf("%w: unsupported password algo %T", command.ErrUnsupported, p.NewAlgo)
	}
	newHash, err := auth.NewPasswordHash([]byte(next), algo)
	if err != nil {
		return false, err
	}
	var oldSRP tg.InputCheckPasswordSRPClass = &tg.InputCheckPasswordEmpty{}
	if p.HasPassword {
		h, err := auth.PasswordHash([]byte(current), p.SRPID, p.SRPB, p.SecureRandom, p.CurrentAlgo)
		if err != nil {
			return true, err
		}
		oldSRP = h
	}
	_, err = api.AccountUpdatePasswordSettings(ctx, &tg.AccountUpdatePasswordSettingsRequest{
		Password: oldSRP,
		NewSettings: tg.AccountPasswordInputSettings{
			NewAlgo:         algo,
			NewPasswordHash: newHash,
			Hint:            hint,
		},
	})
	if err != nil {
		if IsBadPassword(err) {
			return p.HasPassword, fmt.Errorf("%w: %s", session.ErrBadPassword, err.Error())
		}
		return p.HasPassword, err
	}
	return p.HasPassword, nil
}

// DisablePassword removes the existing 2FA password.
func DisablePassword(ctx context.Context, api *tg.Client, current string) error {
	p, err := api.AccountGetPassword(ctx)
	if err != nil {
		return err
	}
	h, err := auth.PasswordHash([]byte(current), p.SRPID, p.SRPB, p.SecureRandom, p.CurrentAlgo)
	if err != nil {
		return err
	}
	_, err = api.AccountUpdatePasswordSettings(ctx, &tg.AccountUpdatePasswordSettingsRequest{
		Password: h,
		NewSettings: tg.AccountPasswordInputSettings{
			NewAlgo:         &tg.PasswordKdfAlgoUnknown{},
			NewPasswordHash: []byte{},
			Hint:            "",
		},
	})
	if err != nil && IsBadPassword(err) {
		return fmt.Errorf("%w: %s", session.ErrBadPassword, err.Error())
	}
	return err
}

// IsBadPassword maps Telegram password RPC strings to the domain error.
func IsBadPassword(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "PASSWORD_HASH_INVALID") ||
		strings.Contains(msg, "SRP_PASSWORD_CHANGED") ||
		strings.Contains(msg, "PASSWORD_MISSING")
}
