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

// Code returns the stable code used in JSON error output. See the ErrorCode
// type for the full enum.
func Code(err error) ErrorCode {
	// Errors arriving via the daemon IPC are already classified by
	// the daemon side; respect the wire code so the daemon and local
	// paths produce identical envelopes.
	var rc RemoteCoded
	if errors.As(err, &rc) {
		if code := rc.RemoteCode(); code != "" {
			return ErrorCode(code)
		}
	}
	// gotd surfaces FLOOD_WAIT as a raw *tgerr.Error in most call
	// paths (ApplyFloodPolicy isn't wired everywhere yet). Normalise
	// here so we classify both the typed *FloodWaitError and the raw
	// form as flood_wait — otherwise the raw form falls all the way
	// to "unknown" and agents lose the most retry-critical signal.
	if _, ok := telegramsession.AsFloodWait(err); ok {
		return CodeFloodWait
	}
	if code, ok := sentinelCode(err); ok {
		return code
	}
	// Classify known raw Telegram RPC errors that the telegram layer returned
	// unwrapped, instead of letting them fall to "unknown".
	if cls, ok := matchRPC(err); ok {
		return cls.code
	}
	return CodeUnknown
}

// sentinelCode maps tg's own error sentinels to their stable code. The second
// return reports whether any sentinel matched; callers fall back to RPC
// classification (then "unknown") when it doesn't. Code and Message share this
// so the JSON code and message always come from the same classification step.
func sentinelCode(err error) (ErrorCode, bool) {
	switch {
	case errors.Is(err, command.ErrUsage), errors.Is(err, config.ErrInvalid):
		return CodeUsage, true
	case errors.Is(err, telegramsession.ErrAuth):
		return CodeAuthRequired, true
	case errors.Is(err, telegrampeer.ErrNotFound):
		return CodePeerNotFound, true
	case errors.Is(err, telegrampeer.ErrForbidden):
		return CodePeerForbidden, true
	case errors.Is(err, telegramsession.ErrFloodWait):
		return CodeFloodWait, true
	case errors.Is(err, telegramsession.ErrNetwork):
		return CodeNetwork, true
	case errors.Is(err, telegramsession.ErrRateExhausted):
		return CodeRateExhausted, true
	case errors.Is(err, command.ErrPrecondition):
		return CodePrecondition, true
	case errors.Is(err, command.ErrUnsupported):
		return CodeUnsupported, true
	case errors.Is(err, account.ErrBusy):
		return CodeBusy, true
	case errors.Is(err, telegrampeer.ErrAmbiguous):
		return CodePeerAmbiguous, true
	case errors.Is(err, telegrammessage.ErrNotFound):
		return CodeMessageNotFound, true
	case errors.Is(err, telegrampeer.ErrCacheMiss):
		return CodeCacheMiss, true
	case errors.Is(err, telegrammessage.ErrNoMedia):
		return CodeNoMedia, true
	case errors.Is(err, telegrammessage.ErrNoLink):
		return CodeNoLink, true
	case errors.Is(err, command.ErrNotConfirmed):
		return CodeNotConfirmed, true
	case errors.Is(err, telegrammessage.ErrRevokeRequired):
		return CodeRevokeReq, true
	case errors.Is(err, telegramsession.ErrCurrent):
		return CodeCurrentSess, true
	case errors.Is(err, telegramchat.ErrInvalidInvite):
		return CodeInvalidInvite, true
	case errors.Is(err, telegramsession.ErrBadPassword):
		return CodeBadPassword, true
	default:
		return CodeUnknown, false
	}
}
