package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"github.com/vika2603/telegram-cli/internal/account"
)

// Run is the sole lifecycle entry: builds a telegram.Client, starts it via
// telegram.Client.Run, and invokes fn inside the Run callback with a live
// directClient. The MTProto connection is torn down when fn returns or ctx is
// cancelled.
//
// A stored StateEXPIRED Meta is rejected before dialing. Once dialed, Self()
// is the liveness probe: auth.IsUnauthorized surfaces as ErrAuth and flips
// Meta to EXPIRED best-effort.
func Run(ctx context.Context, acct *account.Account, opts Options, fn func(ctx context.Context, cl Client) error) error {
	if acct.Meta.State == account.StateEXPIRED {
		return fmt.Errorf("account %s: %w", acct.Meta.Name, ErrAuth)
	}
	b, err := buildTelegramClient(acct, opts)
	if err != nil {
		return err
	}
	defer b.close()
	return stripCallbackPrefix(b.tgCl.Run(ctx, func(ctx context.Context) error {
		return runLifecycle(ctx, acct, opts, b.tgCl, func(self tg.User) Client {
			return &directClient{
				tgCl: b.tgCl,
				self: self,
				opts: opts,
				acct: acct,
			}
		}, fn)
	}))
}

// stripCallbackPrefix unwraps the "callback: <inner>" wrapper that
// gotd's telegram.Client.Run applies (via go-faster/errors.Wrap) to
// every error escaping the user-supplied callback. The prefix is
// structural noise we don't want surfaced to CLI users; sentinel
// errors remain reachable through errors.Is/As regardless of whether
// the wrap is stripped.
func stripCallbackPrefix(err error) error {
	if err == nil {
		return nil
	}
	inner := errors.Unwrap(err)
	if inner == nil {
		return err
	}
	if strings.HasPrefix(err.Error(), "callback: ") && err.Error() == "callback: "+inner.Error() {
		return inner
	}
	return err
}

// selfProbe is the minimum *telegram.Client surface runLifecycle needs. A
// fake implementation lets tests exercise the lifecycle without a real dial.
type selfProbe interface {
	Self(ctx context.Context) (*tg.User, error)
}

// runLifecycle is the testable core of Run: it runs inside tgCl.Run's
// callback, probes Self, handles auth-expiry bookkeeping, constructs the
// per-invocation Client via makeClient, and hands control to fn.
//
// Invariants:
//   - probe.Self auth error: makeClient and fn are not invoked; Meta flips to
//     EXPIRED best-effort and ErrAuth is returned.
//   - probe.Self success: fn is invoked exactly once with a non-nil Client.
//   - ctx cancelled mid-fn: ctx error surfaces (not ErrAuth).
func runLifecycle(
	ctx context.Context,
	acct *account.Account,
	opts Options,
	probe selfProbe,
	makeClient func(self tg.User) Client,
	fn func(context.Context, Client) error,
) error {
	self, err := probe.Self(ctx)
	if err != nil {
		return maybeFlipExpired(acct, opts, err)
	}
	if fnErr := fn(ctx, makeClient(*self)); fnErr != nil {
		return maybeFlipExpired(acct, opts, fnErr)
	}
	return nil
}

// maybeFlipExpired inspects err. If auth.IsUnauthorized, it persists
// State=EXPIRED to Meta and returns ErrAuth. Otherwise it returns err
// unchanged. A WriteMeta failure does not mutate the in-memory Meta (the
// next invocation will retry the transition).
func maybeFlipExpired(acct *account.Account, opts Options, err error) error {
	if !auth.IsUnauthorized(err) {
		return err
	}
	if m := acct.Meta; m.State != account.StateEXPIRED {
		m.State = account.StateEXPIRED
		if werr := account.WriteMeta(m); werr != nil {
			if opts.Logger != nil {
				opts.Logger.Warn("transition to EXPIRED failed",
					zap.String("account", acct.Meta.Name), zap.Error(werr))
			}
		} else {
			acct.Meta = m
		}
	}
	return fmt.Errorf("account %s: %w", acct.Meta.Name, ErrAuth)
}
