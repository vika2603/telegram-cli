package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/daemon"
)

// staleSocketWarnOnce ensures the stale-socket warning fires at most
// once per process, so chained commands (or a single `tg watch` that
// internally re-probes) do not spam stderr with the same message.
var staleSocketWarnOnce sync.Once

// MaybeDialDaemon is the daemon-aware wrapper every command should use
// when it has a fast-path through the per-account daemon socket. It
// returns a live daemon.Client when the socket is reachable and the
// caller has not opted out via --no-daemon; nil otherwise. A nil
// client is the unambiguous signal that the caller should fall back
// to its local MTProto path.
//
// Fallback paths are silent by default — the user expects daemon
// presence to be a performance optimisation, not a behavioural switch.
// The one exception is a stale-socket state: the socket file exists
// but the daemon is not answering, which strongly suggests a crashed
// worker. In that case we write a single warning to stderr (once per
// process) pointing the user at `tg daemon status` / `tg daemon logs`.
//
// The error return is reserved for "tried to dial but the daemon
// declined" scenarios (e.g. schema mismatch). Routine absence — no
// socket file, connection refused — surfaces as (nil, nil) so the
// fallback is automatic.
func MaybeDialDaemon(ctx context.Context, inv *Invocation, acct *account.Account) (*daemon.Client, error) {
	if inv == nil || acct == nil {
		return nil, nil
	}
	if inv.NoDaemon {
		return nil, nil
	}
	switch daemon.DaemonProbe(acct.Meta.Name) {
	case daemon.DaemonStateNotInstalled:
		return nil, nil
	case daemon.DaemonStateStaleSocket:
		warnStaleSocket(inv, acct.Meta.Name)
		return nil, nil
	case daemon.DaemonStateReachable:
		// fallthrough to Dial below
	}
	cl, err := daemon.Dial(ctx, acct.Meta.Name)
	if err != nil {
		// Treat a botched Dial (e.g. socket present but daemon dying
		// between Probe and Dial — a race within the probe window) as
		// "no daemon" rather than a hard error so the command still
		// completes via local MTProto.
		warnStaleSocket(inv, acct.Meta.Name)
		return nil, nil //nolint:nilerr
	}
	return cl, nil
}

func warnStaleSocket(inv *Invocation, account string) {
	if inv == nil || inv.IOStreams == nil || inv.IOStreams.ErrOut == nil {
		return
	}
	staleSocketWarnOnce.Do(func() {
		_, _ = fmt.Fprintf(inv.IOStreams.ErrOut,
			"warning: daemon socket for account %q exists but is unreachable; "+
				"falling back to local MTProto. "+
				"run 'tg daemon status' / 'tg daemon logs' to inspect.\n",
			account)
	})
}
