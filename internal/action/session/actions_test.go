package session

import (
	"context"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/command"
	tgsession "github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func auths() *tg.AccountAuthorizations {
	return &tg.AccountAuthorizations{
		Authorizations: []tg.Authorization{
			{Hash: 100, Current: true, DeviceModel: "Desktop", AppName: "Telegram", DateCreated: 1704067200},
			{Hash: 200, DeviceModel: "iPhone", AppName: "Telegram"},
		},
	}
}

func fetchAuths(a *tg.AccountAuthorizations) FetchFunc {
	return func(context.Context, *tg.Client) (*tg.AccountAuthorizations, error) {
		return a, nil
	}
}

func TestListMapsAuthorizations(t *testing.T) {
	rows, err := List(context.Background(), nil, fetchAuths(auths()))
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, "100", rows[0].Hash)
	require.True(t, rows[0].Current)
	require.Equal(t, "2024-01-01T00:00:00Z", rows[0].DateCreated)
}

func TestValidateTerminate(t *testing.T) {
	err := ValidateTerminate(TerminateRequest{})
	require.ErrorIs(t, err, command.ErrUsage)

	err = ValidateTerminate(TerminateRequest{Hash: "100", AllOthers: true})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestTerminateRejectsCurrentSession(t *testing.T) {
	_, err := Terminate(
		context.Background(),
		nil,
		TerminateRequest{Hash: "100", Prompter: &ui.StubPrompter{Answers: []any{true}}},
		fetchAuths(auths()),
		func(context.Context, *tg.Client, int64) error { return nil },
		func(context.Context, *tg.Client, []int64) error { return nil },
	)
	require.ErrorIs(t, err, tgsession.ErrCurrent)
}

func TestTerminateAllOthersCollectsVictimHashes(t *testing.T) {
	var got []int64
	row, err := Terminate(
		context.Background(),
		nil,
		TerminateRequest{AllOthers: true, Yes: true, Prompter: &ui.StubPrompter{}},
		fetchAuths(auths()),
		func(context.Context, *tg.Client, int64) error { return nil },
		func(_ context.Context, _ *tg.Client, hashes []int64) error {
			got = hashes
			return nil
		},
	)
	require.NoError(t, err)
	require.Equal(t, []int64{200}, got)
	require.Equal(t, true, row["all_others"])
	require.Equal(t, "100", row["kept_hash"])
}
