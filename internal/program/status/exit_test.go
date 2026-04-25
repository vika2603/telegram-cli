package status

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/command"
	telegramchat "github.com/vika2603/telegram-cli/internal/telegram/chat"
	telegrammessage "github.com/vika2603/telegram-cli/internal/telegram/message"
	telegrampeer "github.com/vika2603/telegram-cli/internal/telegram/peer"
	telegramsession "github.com/vika2603/telegram-cli/internal/telegram/session"
)

func TestMapExitCode_allSentinels(t *testing.T) {
	cases := map[error]int{
		nil:                               0,
		command.ErrUsage:                  2,
		telegramsession.ErrAuth:           3,
		telegrampeer.ErrNotFound:          4,
		telegrampeer.ErrForbidden:         5,
		telegramsession.ErrFloodWait:      6,
		telegramsession.ErrNetwork:        7,
		telegramsession.ErrRateExhausted:  6,
		command.ErrPrecondition:           9,
		command.ErrUnsupported:            9,
		account.ErrBusy:                   72,
		telegrampeer.ErrAmbiguous:         4,
		telegrammessage.ErrNotFound:       4,
		telegrampeer.ErrCacheMiss:         9,
		telegrammessage.ErrNoMedia:        9,
		telegrammessage.ErrNoLink:         9,
		command.ErrNotConfirmed:           73,
		telegrammessage.ErrRevokeRequired: 8,
		telegramsession.ErrCurrent:        8,
		telegramchat.ErrInvalidInvite:     8,
		telegramsession.ErrBadPassword:    8,
		errors.New("random"):              1,
	}
	for err, want := range cases {
		require.Equal(t, want, MapExitCode(err), "err=%v", err)
	}
}

func TestMapExitCode_NoResults(t *testing.T) {
	err := command.NewNoResultsError("empty")
	require.Equal(t, 0, MapExitCode(err))
}

func TestMapExitCode_Cancel(t *testing.T) {
	require.Equal(t, 130, MapExitCode(command.ErrCancel))
}

func TestMapExitCode_Silent(t *testing.T) {
	require.Equal(t, 1, MapExitCode(command.ErrSilent))
}

func TestMapExitCode_FlagErrorIsUsage(t *testing.T) {
	require.Equal(t, 2, MapExitCode(command.FlagErrorf("bad")))
}

func TestMapExitCode_Drop3(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{telegramsession.ErrCurrent, 8},
		{telegramchat.ErrInvalidInvite, 8},
		{telegramsession.ErrBadPassword, 8},
		{fmt.Errorf("wrapped: %w", telegramsession.ErrCurrent), 8},
	}
	for _, c := range cases {
		require.Equal(t, c.want, MapExitCode(c.err))
	}
}
