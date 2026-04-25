package auth_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	actionauth "github.com/vika2603/telegram-cli/internal/action/auth"
	"github.com/vika2603/telegram-cli/internal/authflow"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
)

func ptr[T any](v T) *T { return &v }

func TestLogin_InvalidNameIsUsage(t *testing.T) {
	_, err := actionauth.Login(context.Background(), actionauth.LoginRequest{Name: "../bad"}, actionauth.LoginDeps{})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestLogin_NewNoLoginCreatesSlotFromConfig(t *testing.T) {
	cfg := config.Defaults()
	cfg.DefaultAccount = ptr("work")
	cfg.APIID = ptr(123)
	cfg.APIHash = ptr("hash")

	var stored *account.Meta
	doLoginCalled := false
	dto, err := actionauth.Login(context.Background(), actionauth.LoginRequest{
		Name:    "work",
		NoLogin: true,
	}, actionauth.LoginDeps{
		Config: func() (*config.Config, error) { return &cfg, nil },
		ReadMeta: func(name string) (account.Meta, error) {
			if stored == nil {
				return account.Meta{}, account.ErrAccountNotFound
			}
			return *stored, nil
		},
		AddAccount: func(meta account.Meta) error {
			stored = &meta
			return nil
		},
		DoLogin: func(context.Context, authflow.LoginOptions) error {
			doLoginCalled = true
			return nil
		},
	})
	require.NoError(t, err)
	require.False(t, doLoginCalled)
	require.NotNil(t, stored)
	require.Equal(t, 123, stored.APIID)
	require.Equal(t, "hash", stored.APIHash)
	require.True(t, dto.Default)
}

func TestLogin_ForcePassesNormalizedFlagsToDoLogin(t *testing.T) {
	cfg := config.Defaults()
	got := authflow.LoginOptions{}
	_, err := actionauth.Login(context.Background(), actionauth.LoginRequest{
		Name:           "work",
		Force:          true,
		APIID:          99,
		APIHash:        "rotated",
		APIIDChanged:   true,
		APIHashChanged: true,
	}, actionauth.LoginDeps{
		Config: func() (*config.Config, error) { return &cfg, nil },
		ReadMeta: func(string) (account.Meta, error) {
			return account.Meta{Name: "work", State: account.StateAUTHED}, nil
		},
		AddAccount: func(account.Meta) error { return nil },
		DoLogin: func(_ context.Context, opts authflow.LoginOptions) error {
			got = opts
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, "work", got.Name)
	require.True(t, got.Force)
	require.Equal(t, 99, got.APIID)
	require.Equal(t, "rotated", got.APIHash)
	require.True(t, got.APIIDChanged)
	require.True(t, got.APIHashChanged)
}

func TestStatus_ProbeSkippedForNewSlot(t *testing.T) {
	result, err := actionauth.Status(context.Background(), actionauth.StatusRequest{
		Name:  "work",
		Probe: true,
	}, actionauth.StatusDeps{
		ResolveAccount: func(string) (*account.Account, error) {
			return &account.Account{Meta: account.Meta{Name: "work", State: account.StateNEW}}, nil
		},
		SessionModified: func(string) (string, error) { return "", nil },
		Probe: func(context.Context, *account.Account) error {
			t.Fatal("probe must not run for NEW slot")
			return nil
		},
	})
	require.NoError(t, err)
	require.True(t, result.Row.Probed)
	require.False(t, result.Row.ProbeOK)
	require.True(t, result.ProbeSkipped)
	require.Equal(t, "NEW", result.ProbeSkippedState)
}

func TestSwitch_NotFoundIsUsage(t *testing.T) {
	_, err := actionauth.Switch(actionauth.SwitchRequest{Name: "ghost"}, actionauth.SwitchDeps{
		ReadMeta:   func(string) (account.Meta, error) { return account.Meta{}, account.ErrAccountNotFound },
		SetDefault: func(string) error { return nil },
	})
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestLogout_NonAuthedPurgeReturnsDefaultClearWarning(t *testing.T) {
	cfg := config.Defaults()
	cfg.DefaultAccount = ptr("work")
	meta := account.Meta{Name: "work", State: account.StateEXPIRED}

	result, err := actionauth.Logout(context.Background(), actionauth.LogoutRequest{
		Name:  "work",
		Purge: true,
		Yes:   true,
	}, actionauth.LogoutDeps{
		ResolveAccount: func(string) (*account.Account, error) {
			return &account.Account{Meta: meta}, nil
		},
		Config:        func() (*config.Config, error) { return &cfg, nil },
		Confirm:       func(string, bool) error { return nil },
		DeleteSession: func(string) error { return nil },
		WriteMeta: func(next account.Meta) error {
			meta = next
			return nil
		},
		RemoveAccount: func(string) error { return nil },
		ClearDefault: func() (actionauth.ClearDefaultResult, error) {
			return actionauth.ClearDefaultResult{Path: "/tmp/config.toml"}, errors.New("denied")
		},
		Do: func(context.Context, actionauth.LogoutDoArgs) error { return nil },
	})
	require.NoError(t, err)
	require.Equal(t, account.StateNEW, meta.State)
	require.False(t, result.Row.DefaultCleared)
	require.Len(t, result.Warnings, 1)
	require.Contains(t, result.Warnings[0], "could not clear default_account")
}
