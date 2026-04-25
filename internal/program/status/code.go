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

// Code returns the stable string code used in JSON error output.
func Code(err error) string {
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
