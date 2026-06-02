package status

import (
	"errors"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	telegramchat "github.com/vika2603/telegram-cli/internal/telegram/chat"
	telegrammessage "github.com/vika2603/telegram-cli/internal/telegram/message"
	telegrampeer "github.com/vika2603/telegram-cli/internal/telegram/peer"
	telegramsession "github.com/vika2603/telegram-cli/internal/telegram/session"
)

// RemoteExitCoded is satisfied by errors that carry a wire-shape exit
// code (e.g. daemon.RemoteError). Same rationale as RemoteCoded in
// code.go — IPC-routed errors have already been mapped by the remote
// side and we honor that mapping verbatim.
type RemoteExitCoded interface {
	RemoteExitCode() int
}

// MapExitCode is the single source of truth for error → exit code translation.
// Unknown errors map to 1.
func MapExitCode(err error) int {
	// Errors received over the daemon IPC carry their final exit code
	// from the remote side; trust it so daemon-routed and locally-run
	// commands produce identical exit codes for the same underlying
	// failure.
	if err != nil {
		var rc RemoteExitCoded
		if errors.As(err, &rc) {
			// Guard against a 0 leaking out of an error frame — a
			// non-nil error must never map to the success exit code.
			// Fall through to the normal mapping (default 1) if the
			// remote side somehow reported 0.
			if ec := rc.RemoteExitCode(); ec > 0 {
				return ec
			}
		}
		// Raw gotd FLOOD_WAIT — same rationale as in code.go.
		if _, ok := telegramsession.AsFloodWait(err); ok {
			return 6
		}
	}
	switch {
	case err == nil:
		return 0
	case errors.Is(err, &command.NoResultsError{}):
		return 0
	case errors.Is(err, command.ErrCancel):
		return 130
	case errors.Is(err, command.ErrSilent):
		return 1
	case errors.Is(err, command.ErrUsage), errors.Is(err, config.ErrInvalid):
		return 2
	case errors.Is(err, telegramsession.ErrAuth):
		return 3
	case errors.Is(err, telegrampeer.ErrNotFound),
		errors.Is(err, telegrampeer.ErrAmbiguous),
		errors.Is(err, telegrammessage.ErrNotFound):
		return 4
	case errors.Is(err, telegrampeer.ErrForbidden):
		return 5
	case errors.Is(err, telegramsession.ErrFloodWait),
		errors.Is(err, telegramsession.ErrRateExhausted):
		return 6
	case errors.Is(err, telegramsession.ErrNetwork):
		return 7
	case errors.Is(err, telegrammessage.ErrRevokeRequired),
		errors.Is(err, telegramsession.ErrCurrent),
		errors.Is(err, telegramchat.ErrInvalidInvite),
		errors.Is(err, telegramsession.ErrBadPassword):
		return 8
	case errors.Is(err, command.ErrPrecondition), errors.Is(err, telegrampeer.ErrCacheMiss):
		return 9
	case errors.Is(err, command.ErrUnsupported),
		errors.Is(err, telegrammessage.ErrNoMedia),
		errors.Is(err, telegrammessage.ErrNoLink):
		return 9
	case errors.Is(err, account.ErrBusy):
		return 72
	case errors.Is(err, command.ErrNotConfirmed):
		return 73
	default:
		// Classify known raw Telegram RPC errors so they map to a stable
		// exit code instead of the catch-all 1.
		if cls, ok := matchRPC(err); ok {
			return cls.exit
		}
		return 1
	}
}
