package terminate_test

import (
	"context"
	"testing"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/cli/session/terminate"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// stubWithPeers replaces f.WithPeers so the callback is called with a nil
// *tg.Client (safe because tests stub Fetch/Reset/ResetAll directly).
func stubWithPeers(f *runtime.Invocation) {
	f.WithPeers = func(
		ctx context.Context,
		_ *account.Account,
		_ session.Options,
		fn func(context.Context, *tg.Client, *peers.Manager, *peer.Resolver) error,
	) error {
		return fn(ctx, nil, nil, nil)
	}
}

func stubAccount(f *runtime.Invocation) {
	f.Account = func(_ string) (*account.Account, error) {
		return &account.Account{}, nil
	}
}

// twoSessions returns a stub with two sessions: hash 100 is current, hash 200 is not.
func twoSessions() *tg.AccountAuthorizations {
	return &tg.AccountAuthorizations{
		Authorizations: []tg.Authorization{
			{Hash: 100, Current: true, DeviceModel: "Desktop", Platform: "Linux", AppName: "Telegram", Country: "US"},
			{Hash: 200, Current: false, DeviceModel: "iPhone", Platform: "iOS", AppName: "Telegram", Country: "DE"},
		},
	}
}

func noopFetch(auths *tg.AccountAuthorizations) func(context.Context, *tg.Client) (*tg.AccountAuthorizations, error) {
	return func(_ context.Context, _ *tg.Client) (*tg.AccountAuthorizations, error) {
		return auths, nil
	}
}

func TestNew_FlagParsing_Hash(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	var captured *terminate.Options
	cmd := terminate.New(f, func(o *terminate.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"12345"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, "12345", captured.Hash)
	require.False(t, captured.AllOthers)
}

func TestNew_FlagParsing_AllOthers(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	var captured *terminate.Options
	cmd := terminate.New(f, func(o *terminate.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{"--all-others"})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.True(t, captured.AllOthers)
	require.Empty(t, captured.Hash)
}

func TestRun_MissingHashAndAllOthers_IsUsage(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	opts := &terminate.Options{
		F:        f,
		Fetch:    noopFetch(twoSessions()),
		Reset:    func(_ context.Context, _ *tg.Client, _ int64) error { return nil },
		ResetAll: func(_ context.Context, _ *tg.Client, _ []int64) error { return nil },
	}
	err := terminate.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
	require.ErrorContains(t, err, "provide a hash or --all-others")
}

func TestRun_HashAndAllOthers_IsUsage(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	opts := &terminate.Options{
		F:         f,
		Hash:      "200",
		AllOthers: true,
		Fetch:     noopFetch(twoSessions()),
		Reset:     func(_ context.Context, _ *tg.Client, _ int64) error { return nil },
		ResetAll:  func(_ context.Context, _ *tg.Client, _ []int64) error { return nil },
	}
	err := terminate.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
	require.ErrorContains(t, err, "cannot use <hash> and --all-others together")
}

func TestRun_HashNotFound(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)
	opts := &terminate.Options{
		F:        f,
		Hash:     "999",
		Fetch:    noopFetch(twoSessions()),
		Reset:    func(_ context.Context, _ *tg.Client, _ int64) error { return nil },
		ResetAll: func(_ context.Context, _ *tg.Client, _ []int64) error { return nil },
	}
	err := terminate.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
	require.ErrorContains(t, err, "no session with hash")
}

func TestRun_HashIsCurrent(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)
	opts := &terminate.Options{
		F:        f,
		Hash:     "100", // hash 100 is current
		Fetch:    noopFetch(twoSessions()),
		Reset:    func(_ context.Context, _ *tg.Client, _ int64) error { return nil },
		ResetAll: func(_ context.Context, _ *tg.Client, _ []int64) error { return nil },
	}
	err := terminate.Run(context.Background(), opts)
	require.ErrorIs(t, err, session.ErrCurrent)
}

func TestRun_DeclinedPrompt(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)
	f.Prompter = &ui.StubPrompter{Answers: []any{false}}
	opts := &terminate.Options{
		F:        f,
		Hash:     "200", // non-current session
		Fetch:    noopFetch(twoSessions()),
		Reset:    func(_ context.Context, _ *tg.Client, _ int64) error { return nil },
		ResetAll: func(_ context.Context, _ *tg.Client, _ []int64) error { return nil },
	}
	err := terminate.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestRun_AcceptedTerminatesOne(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)
	f.Prompter = &ui.StubPrompter{Answers: []any{true}}

	var resetHash int64
	opts := &terminate.Options{
		F:     f,
		Hash:  "200",
		Fetch: noopFetch(twoSessions()),
		Reset: func(_ context.Context, _ *tg.Client, h int64) error {
			resetHash = h
			return nil
		},
		ResetAll: func(_ context.Context, _ *tg.Client, _ []int64) error { return nil },
	}
	require.NoError(t, terminate.Run(context.Background(), opts))
	require.Equal(t, int64(200), resetHash)
}

func TestRun_AllOthersCountsNonCurrent(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)
	f.Prompter = &ui.StubPrompter{Answers: []any{true}}

	auths := &tg.AccountAuthorizations{
		Authorizations: []tg.Authorization{
			{Hash: 100, Current: true, DeviceModel: "Desktop", Platform: "Linux", AppName: "Telegram"},
			{Hash: 200, Current: false, DeviceModel: "iPhone", Platform: "iOS", AppName: "Telegram"},
			{Hash: 300, Current: false, DeviceModel: "Android", Platform: "Android", AppName: "Telegram"},
		},
	}

	var resetHashes []int64
	opts := &terminate.Options{
		F:         f,
		AllOthers: true,
		Fetch:     noopFetch(auths),
		Reset:     func(_ context.Context, _ *tg.Client, _ int64) error { return nil },
		ResetAll: func(_ context.Context, _ *tg.Client, hashes []int64) error {
			resetHashes = hashes
			return nil
		},
	}
	require.NoError(t, terminate.Run(context.Background(), opts))
	require.Len(t, resetHashes, 2)
}

func TestRun_AllOthers_NoOthers(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)

	auths := &tg.AccountAuthorizations{
		Authorizations: []tg.Authorization{
			{Hash: 100, Current: true, DeviceModel: "Desktop", Platform: "Linux", AppName: "Telegram"},
		},
	}
	opts := &terminate.Options{
		F:         f,
		AllOthers: true,
		Fetch:     noopFetch(auths),
		Reset:     func(_ context.Context, _ *tg.Client, _ int64) error { return nil },
		ResetAll:  func(_ context.Context, _ *tg.Client, _ []int64) error { return nil },
	}
	err := terminate.Run(context.Background(), opts)
	// NoResultsError exits 0; errors.Is must recognise it via pointer type check.
	require.Error(t, err)
	var nre *command.NoResultsError
	require.ErrorAs(t, err, &nre)
	require.Contains(t, err.Error(), "no other sessions")
}
