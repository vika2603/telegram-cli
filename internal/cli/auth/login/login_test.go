package login_test

import (
	"context"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/authflow"
	"github.com/vika2603/telegram-cli/internal/cli/auth/login"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// ptr is a local helper for building *T values inline.
func ptr[T any](v T) *T { return &v }

// newInvocationWithCreds returns a test invocation whose Config includes an
// api_id/api_hash so slot creation does not fall back to interactive prompts.
func newInvocationWithCreds(t *testing.T) *runtime.Invocation {
	t.Helper()
	f := runtime.NewTestInvocation(t)
	f.Config = func() (*config.Config, error) {
		d := config.Defaults()
		d.APIID = ptr(12345)
		d.APIHash = ptr("testhash")
		return &d, nil
	}
	return f
}

// TestNew_FlagParsing verifies that all flags are wired correctly by executing
// the cobra command with a runF capture function.
func TestNew_FlagParsing(t *testing.T) {
	var captured *login.Options
	f := runtime.NewTestInvocation(t)
	cmd := login.New(f, func(o *login.Options) error {
		captured = o
		return nil
	})
	cmd.SetArgs([]string{
		"work",
		"--force",
		"--qr",
		"--no-login",
		"--api-id", "99",
		"--api-hash", "myhash",
	})
	require.NoError(t, cmd.Execute())
	require.NotNil(t, captured)
	require.Equal(t, "work", captured.Name)
	require.True(t, captured.Force)
	require.True(t, captured.QR)
	require.True(t, captured.NoLogin)
	require.Equal(t, 99, captured.APIID)
	require.Equal(t, "myhash", captured.APIHash)
}

// TestRun_InvalidName verifies that an account name that fails IsValidName
// returns ErrUsage.
func TestRun_InvalidName(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	f := runtime.NewTestInvocation(t)
	opts := &login.Options{
		Name:    "../../etc/passwd",
		NoLogin: true,
		F:       f,
		DoLogin: func(_ context.Context, _ authflow.LoginOptions) error {
			t.Fatal("DoLogin must not be called for invalid names")
			return nil
		},
	}
	err := login.Run(context.Background(), &cobra.Command{}, opts)
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
}

// TestRun_NewSlot_NoLogin_CreatesOnly verifies that --no-login creates the
// account slot in state NEW and emits the DTO without logging in.
func TestRun_NewSlot_NoLogin_CreatesOnly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	f := newInvocationWithCreds(t)

	doLoginCalled := false
	opts := &login.Options{
		Name:    "testslot",
		NoLogin: true,
		F:       f,
		DoLogin: func(_ context.Context, _ authflow.LoginOptions) error {
			doLoginCalled = true
			return nil
		},
	}

	err := login.Run(context.Background(), &cobra.Command{}, opts)
	require.NoError(t, err)
	require.False(t, doLoginCalled, "DoLogin must not be called with --no-login")

	// Verify slot was created with state NEW.
	meta, err := account.ReadMeta("testslot")
	require.NoError(t, err)
	require.Equal(t, account.StateNEW, meta.State)
}

// TestRun_ExistingAUTHED_NoForce_IsNoop verifies that an already-AUTHED
// account without --force emits the DTO and does NOT invoke DoLogin.
func TestRun_ExistingAUTHED_NoForce_IsNoop(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	f := newInvocationWithCreds(t)

	// Pre-create AUTHED account slot.
	require.NoError(t, account.AddAccount(account.Meta{
		Name:    "authedacct",
		State:   account.StateAUTHED,
		APIID:   12345,
		APIHash: "testhash",
	}))

	doLoginCalled := false
	opts := &login.Options{
		Name:  "authedacct",
		Force: false,
		F:     f,
		DoLogin: func(_ context.Context, _ authflow.LoginOptions) error {
			doLoginCalled = true
			return nil
		},
	}

	err := login.Run(context.Background(), &cobra.Command{}, opts)
	require.NoError(t, err)
	require.False(t, doLoginCalled, "DoLogin must not be called for AUTHED account without --force")
}

// TestRun_ExistingAUTHED_Force_RerunsLogin verifies that --force causes
// DoLogin to be invoked even when the account is already AUTHED.
func TestRun_ExistingAUTHED_Force_RerunsLogin(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	f := newInvocationWithCreds(t)

	// Pre-create AUTHED account slot.
	require.NoError(t, account.AddAccount(account.Meta{
		Name:    "authedforce",
		State:   account.StateAUTHED,
		APIID:   12345,
		APIHash: "testhash",
	}))

	doLoginCalled := false
	opts := &login.Options{
		Name:  "authedforce",
		Force: true,
		F:     f,
		DoLogin: func(_ context.Context, _ authflow.LoginOptions) error {
			doLoginCalled = true
			// Simulate a successful login by leaving state intact (real DoLogin
			// would update state; here we just confirm it was called).
			return nil
		},
	}

	err := login.Run(context.Background(), &cobra.Command{}, opts)
	require.NoError(t, err)
	require.True(t, doLoginCalled, "DoLogin must be called for AUTHED account with --force")
}

// TestRun_ExistingAUTHED_Force_DoLoginError verifies that errors from
// DoLogin are propagated back to the caller.
func TestRun_ExistingAUTHED_Force_DoLoginError(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	f := newInvocationWithCreds(t)

	require.NoError(t, account.AddAccount(account.Meta{
		Name:    "failacct",
		State:   account.StateAUTHED,
		APIID:   12345,
		APIHash: "testhash",
	}))

	sentinel := errors.New("login failed")
	opts := &login.Options{
		Name:  "failacct",
		Force: true,
		F:     f,
		DoLogin: func(_ context.Context, _ authflow.LoginOptions) error {
			return sentinel
		},
	}

	err := login.Run(context.Background(), &cobra.Command{}, opts)
	require.ErrorIs(t, err, sentinel)
}
