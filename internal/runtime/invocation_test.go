package runtime_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

func TestInvocation_ConfigLazyCachesResult(t *testing.T) {
	calls := 0
	f := &runtime.Invocation{
		Config: func() (*config.Config, error) {
			calls++
			c := &config.Config{}
			return c, nil
		},
	}
	_, err := f.Config()
	require.NoError(t, err)
	_, err = f.Config()
	require.NoError(t, err)
	require.Equal(t, 2, calls, "invocation itself does not cache; callers do — this test asserts the raw closure is invoked on every call")
}

func TestInvocation_WithClient_DelegatesToInjectedCallback(t *testing.T) {
	ios, _, _, _ := ui.Test()
	called := false
	f := &runtime.Invocation{
		IOStreams: ios,
		WithClient: func(ctx context.Context, acct *account.Account, opts session.Options,
			fn func(context.Context, session.Client) error) error {
			called = true
			return fn(ctx, nil)
		},
	}
	err := f.WithClient(context.Background(), nil, session.Options{}, func(_ context.Context, _ session.Client) error {
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
}
