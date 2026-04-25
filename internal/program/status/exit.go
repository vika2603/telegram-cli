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

// MapExitCode is the single source of truth for error → exit code translation.
// Unknown errors map to 1.
func MapExitCode(err error) int {
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
		return 1
	}
}
