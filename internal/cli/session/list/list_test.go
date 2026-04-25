package list_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/cli/session/list"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/output"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestNew_FlagParsing(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	var captured *list.Options
	cmd := list.New(f, func(o *list.Options) error { captured = o; return nil })
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
}

func TestRun_NilFetchReturnsPrecondition(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	opts := &list.Options{F: f}
	err := list.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
	require.ErrorContains(t, err, "session list fetch function is not configured")
}

func TestRun_AccountErrorPropagated(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	// Default invocation Account returns ErrPrecondition; Fetch set so we reach Account().
	opts := &list.Options{
		F: f,
		Fetch: func(_ context.Context, _ *tg.Client) (*tg.AccountAuthorizations, error) {
			return nil, nil
		},
	}
	err := list.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_WritesRowsFromFetch(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios

	// Wire Account to succeed.
	f.Account = func(_ string) (*account.Account, error) {
		return &account.Account{}, nil
	}

	// Wire WithPeers to invoke the callback with a nil api (Fetch is stubbed).
	f.WithPeers = func(
		ctx context.Context,
		_ *account.Account,
		_ session.Options,
		fn func(context.Context, *tg.Client, *peers.Manager, *peer.Resolver) error,
	) error {
		return fn(ctx, nil, nil, nil)
	}

	stubAuths := &tg.AccountAuthorizations{
		Authorizations: []tg.Authorization{
			{
				Hash:        111,
				Current:     true,
				DeviceModel: "Desktop",
				Platform:    "Linux",
				AppName:     "Telegram",
				DateCreated: 1704067200, // 2024-01-01T00:00:00Z
				DateActive:  1717243200, // 2024-06-01T12:00:00Z (approx)
			},
			{
				Hash:        222,
				Current:     false,
				DeviceModel: "iPhone",
				Platform:    "iOS",
				AppName:     "Telegram",
				DateCreated: 1684137600,
				DateActive:  1710936000,
			},
		},
	}

	opts := &list.Options{
		F: f,
		Fetch: func(_ context.Context, _ *tg.Client) (*tg.AccountAuthorizations, error) {
			return stubAuths, nil
		},
	}
	require.NoError(t, list.Run(context.Background(), opts))

	lines := strings.Split(strings.TrimRight(stdout.String(), "\n"), "\n")
	require.Len(t, lines, 2)

	var row0 output.AccountSessionRow
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &row0))
	require.Equal(t, "111", row0.Hash)
	require.True(t, row0.Current)
	require.Equal(t, "Desktop", row0.DeviceModel)
	require.Equal(t, "2024-01-01T00:00:00Z", row0.DateCreated)

	var row1 output.AccountSessionRow
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &row1))
	require.Equal(t, "222", row1.Hash)
	require.False(t, row1.Current)
	require.Equal(t, "iPhone", row1.DeviceModel)
}

func TestRun_EmptyAuthorizationList(t *testing.T) {
	ios, _, stdout, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios

	f.Account = func(_ string) (*account.Account, error) {
		return &account.Account{}, nil
	}
	f.WithPeers = func(
		ctx context.Context,
		_ *account.Account,
		_ session.Options,
		fn func(context.Context, *tg.Client, *peers.Manager, *peer.Resolver) error,
	) error {
		return fn(ctx, nil, nil, nil)
	}

	opts := &list.Options{
		F: f,
		Fetch: func(_ context.Context, _ *tg.Client) (*tg.AccountAuthorizations, error) {
			return &tg.AccountAuthorizations{Authorizations: nil}, nil
		},
	}
	require.NoError(t, list.Run(context.Background(), opts))
	require.Empty(t, stdout.String())
}
