package session

import (
	"context"
	"fmt"

	"github.com/gotd/td/telegram/auth"
	"go.uber.org/zap"

	"github.com/vika2603/telegram-cli/internal/account"
)

// Login drives the code-login flow (phone + SMS/app + 2FA). It constructs a
// telegram.Client internally, runs auth.Flow inside its Run callback, and on
// success the session bytes are persisted by the session storage adapter.
// Post-auth bookkeeping:
//   - flip Meta.State to AUTHED
//   - persist Meta.Phone from Self() in E.164 form
//   - cache the self peer in peers.db (non-fatal on failure)
//
// All post-auth writes happen only AFTER auth.IfNecessary succeeds. A failed
// login never mutates Meta or peers.db. A self-cache write failure logs a
// warn but does not fail the login.
func Login(ctx context.Context, acct *account.Account, opts Options, authr account.UserAuthenticator) error {
	b, err := buildTelegramClient(acct, opts, dbPeers)
	if err != nil {
		return err
	}
	defer b.close()
	return b.tgCl.Run(ctx, func(ctx context.Context) error {
		adapter := newAuthAdapter(authr, opts.Logger)
		flow := auth.NewFlow(adapter, auth.SendCodeOptions{})
		if err := b.tgCl.Auth().IfNecessary(ctx, flow); err != nil {
			return fmt.Errorf("auth flow: %w", err)
		}
		self, err := b.tgCl.Self(ctx)
		if err != nil {
			return fmt.Errorf("resolve self after auth: %w", err)
		}
		m := acct.Meta
		m.State = account.StateAUTHED
		if self.Phone != "" {
			m.Phone = "+" + self.Phone
		}
		if werr := account.WriteMeta(m); werr != nil {
			return fmt.Errorf("persist AUTHED: %w", werr)
		}
		acct.Meta = m
		if b.pStore != nil {
			if cerr := b.pStore.CacheSelf(self); cerr != nil && opts.Logger != nil {
				opts.Logger.Warn("cache self peer after login",
					zap.String("account", acct.Meta.Name), zap.Error(cerr))
			}
		}
		return nil
	})
}
