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

// RemoteCoded is satisfied by errors that already carry a wire-shape
// code (e.g. daemon.RemoteError received from an IPC call). The local
// path's type switch can't classify them via errors.Is because the
// local sentinels never get embedded in a remote error chain — the
// code is just a string the remote side computed. Honor it directly.
type RemoteCoded interface {
	RemoteCode() string
}

// Code returns the stable string code used in JSON error output.
func Code(err error) string {
	// Errors arriving via the daemon IPC are already classified by
	// the daemon side; respect the wire code so the daemon and local
	// paths produce identical envelopes.
	var rc RemoteCoded
	if errors.As(err, &rc) {
		if code := rc.RemoteCode(); code != "" {
			return code
		}
	}
	// gotd surfaces FLOOD_WAIT as a raw *tgerr.Error in most call
	// paths (ApplyFloodPolicy isn't wired everywhere yet). Normalise
	// here so we classify both the typed *FloodWaitError and the raw
	// form as flood_wait — otherwise the raw form falls all the way
	// to "unknown" and agents lose the most retry-critical signal.
	if _, ok := telegramsession.AsFloodWait(err); ok {
		return "flood_wait"
	}
	switch {
	case errors.Is(err, command.ErrUsage), errors.Is(err, config.ErrInvalid):
		return "usage"
	case errors.Is(err, telegramsession.ErrAuth):
		return "auth_required"
	case errors.Is(err, telegrampeer.ErrNotFound):
		return "peer_not_found"
	case errors.Is(err, telegrampeer.ErrForbidden):
		return "peer_forbidden"
	case errors.Is(err, telegramsession.ErrFloodWait):
		return "flood_wait"
	case errors.Is(err, telegramsession.ErrNetwork):
		return "network"
	case errors.Is(err, telegramsession.ErrRateExhausted):
		return "rate_exhausted"
	case errors.Is(err, command.ErrPrecondition):
		return "precondition"
	case errors.Is(err, command.ErrUnsupported):
		return "unsupported"
	case errors.Is(err, account.ErrBusy):
		return "busy"
	case errors.Is(err, telegrampeer.ErrAmbiguous):
		return "peer_ambiguous"
	case errors.Is(err, telegrammessage.ErrNotFound):
		return "message_not_found"
	case errors.Is(err, telegrampeer.ErrCacheMiss):
		return "cache_miss"
	case errors.Is(err, telegrammessage.ErrNoMedia):
		return "no_media"
	case errors.Is(err, telegrammessage.ErrNoLink):
		return "no_link"
	case errors.Is(err, command.ErrNotConfirmed):
		return "not_confirmed"
	case errors.Is(err, telegrammessage.ErrRevokeRequired):
		return "revoke_required"
	case errors.Is(err, telegramsession.ErrCurrent):
		return "current_session"
	case errors.Is(err, telegramchat.ErrInvalidInvite):
		return "invalid_invite"
	case errors.Is(err, telegramsession.ErrBadPassword):
		return "bad_password"
	default:
		return "unknown"
	}
}
