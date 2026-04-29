package session

import (
	"context"
	"errors"
	"fmt"

	gotdauth "github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"

	"github.com/vika2603/telegram-cli/internal/account"
)

// LoginQR drives QR login. gotd's QR flow needs a LoggedIn channel fed by the
// server-side login-token update; that update only lands via an UpdateHandler,
// so we install a tg.UpdateDispatcher on the client before Run and register
// qrlogin.OnLoginToken against it.
//
// When the account has 2FA enabled, AuthImportLoginToken returns
// SESSION_PASSWORD_NEEDED; in that case authr.Password is used to complete the
// flow via auth.Client.Password. Pass authr=nil only in contexts where 2FA is
// known not to apply.
//
// Post-auth bookkeeping mirrors Login:
//   - flip Meta.State to AUTHED
//   - persist Meta.Phone from Self() in E.164 form
//
// display.Done(true) is called only after all post-auth bookkeeping succeeds;
// any earlier failure fires display.Done(false) so the UI can clean up.
func LoginQR(ctx context.Context, acct *account.Account, opts Options, authr account.UserAuthenticator, display account.QRDisplay) error {
	dispatcher := tg.NewUpdateDispatcher()
	loggedIn := qrlogin.OnLoginToken(&dispatcher)

	b, err := buildTelegramClientWithHandler(acct, opts, &dispatcher)
	if err != nil {
		return err
	}
	defer b.close()

	return b.tgCl.Run(ctx, func(ctx context.Context) error {
		q := b.tgCl.QR()
		firstToken := true
		_, err := q.Auth(ctx, loggedIn, func(ctx context.Context, t qrlogin.Token) error {
			if firstToken {
				firstToken = false
				return display.Show(ctx, t.URL())
			}
			return display.Refresh(ctx, t.URL())
		})
		if err != nil {
			if !tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
				display.Done(ctx, false)
				return fmt.Errorf("qr auth: %w", err)
			}
			if authr == nil {
				display.Done(ctx, false)
				return fmt.Errorf("qr auth: %w: 2FA required but no password source (set TG_2FA_PASSWORD or run from a TTY)", ErrAuth)
			}
			pw, perr := authr.Password(ctx)
			if perr != nil {
				display.Done(ctx, false)
				return fmt.Errorf("qr auth: read 2FA password: %w", perr)
			}
			if _, perr := b.tgCl.Auth().Password(ctx, pw); perr != nil {
				display.Done(ctx, false)
				if errors.Is(perr, gotdauth.ErrPasswordInvalid) {
					return fmt.Errorf("qr auth: %w", ErrBadPassword)
				}
				return fmt.Errorf("qr auth: 2FA: %w", perr)
			}
		}
		self, err := b.tgCl.Self(ctx)
		if err != nil {
			display.Done(ctx, false)
			return fmt.Errorf("resolve self after qr auth: %w", err)
		}
		m := acct.Meta
		m.State = account.StateAUTHED
		if self.Phone != "" {
			m.Phone = "+" + self.Phone
		}
		if werr := account.WriteMeta(m); werr != nil {
			display.Done(ctx, false)
			return fmt.Errorf("persist AUTHED: %w", werr)
		}
		acct.Meta = m
		display.Done(ctx, true)
		return nil
	})
}
