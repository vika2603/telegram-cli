package session

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/auth/qrlogin"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"github.com/vika2603/telegram-cli/internal/account"
)

// LoginQR drives QR login. gotd's QR flow needs a LoggedIn channel fed by the
// server-side login-token update; that update only lands via an UpdateHandler,
// so we install a tg.UpdateDispatcher on the client before Run and register
// qrlogin.OnLoginToken against it.
//
// Post-auth bookkeeping mirrors Login:
//   - flip Meta.State to AUTHED
//   - persist Meta.Phone from Self() in E.164 form
//   - cache self into peers.db (non-fatal on failure)
//
// display.Done(true) is called only after all post-auth bookkeeping succeeds;
// any earlier failure fires display.Done(false) so the UI can clean up.
// Self-cache failure does NOT downgrade Done to false.
func LoginQR(ctx context.Context, acct *account.Account, opts Options, display account.QRDisplay) error {
	dispatcher := tg.NewUpdateDispatcher()
	loggedIn := qrlogin.OnLoginToken(&dispatcher)

	b, err := buildTelegramClientWithHandler(acct, opts, dbPeers, &dispatcher)
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
			display.Done(ctx, false)
			return fmt.Errorf("qr auth: %w", err)
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
		if b.pStore != nil {
			if cerr := b.pStore.CacheSelf(self); cerr != nil && opts.Logger != nil {
				opts.Logger.Warn("cache self peer after qr login",
					zap.String("account", acct.Meta.Name), zap.Error(cerr))
			}
		}
		display.Done(ctx, true)
		return nil
	})
}
