package runtime

import (
	"context"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/daemon"
)

// MaybeDialDaemon is the daemon-aware wrapper every command should use
// when it has a fast-path through the per-account daemon socket. It
// returns a live daemon.Client when the socket is reachable and the
// caller has not opted out via --no-daemon; nil otherwise. A nil
// client is the unambiguous signal that the caller should fall back
// to its local MTProto path.
//
// The error return is reserved for "tried to dial but the daemon
// declined" scenarios (e.g. schema mismatch). Routine absence — no
// socket file, connection refused — surfaces as (nil, nil) so the
// fallback is automatic and silent.
func MaybeDialDaemon(ctx context.Context, inv *Invocation, acct *account.Account) (*daemon.Client, error) {
	if inv == nil || acct == nil {
		return nil, nil
	}
	if inv.NoDaemon {
		return nil, nil
	}
	if !daemon.DaemonReachable(acct.Meta.Name) {
		return nil, nil
	}
	cl, err := daemon.Dial(ctx, acct.Meta.Name)
	if err != nil {
		// Treat a botched Dial (e.g. socket present but daemon dying)
		// as "no daemon" rather than a hard error — local mode still
		// works, surfacing the dial error would prevent the command
		// from running at all.
		return nil, nil //nolint:nilerr
	}
	return cl, nil
}
