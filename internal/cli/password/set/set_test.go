package set_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/gotd/td/telegram/peers"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/cli/password/set"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/peer"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// stubWithPeers wires the invocation's WithPeers so it directly invokes fn with
// nil api/manager/resolver. Tests that stub Probe/Apply don't use these
// parameters, so nil is safe.
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

// stubAccount wires the invocation's Account so it returns a non-nil account.
func stubAccount(f *runtime.Invocation) {
	f.Account = func(_ string) (*account.Account, error) {
		return &account.Account{}, nil
	}
}

func TestNew_FlagParsing(t *testing.T) {
	var captured *set.Options
	f := runtime.NewTestInvocation(t)
	cmd := set.New(f, func(o *set.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{
		"--hint", "myhint",
		"--recovery-email", "me@example.com",
		"--new-stdin",
		"--current-stdin",
	})
	require.NoError(t, cmd.Execute())
	require.Equal(t, "myhint", captured.Hint)
	require.Equal(t, "me@example.com", captured.RecoveryEmail)
	require.True(t, captured.NewStdin)
	require.True(t, captured.CurrentStdin)
}

func TestRun_NilProbeReturnsPrecondition(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)

	opts := &set.Options{
		F:     f,
		Check: nil,
		Apply: func(_ context.Context, _, _, _ string) (bool, error) { return false, nil },
	}
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_NilApplyReturnsPrecondition(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)

	opts := &set.Options{
		F:     f,
		Check: func(_ context.Context) (bool, error) { return false, nil },
		Apply: nil,
	}
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_StdinNewPassword(t *testing.T) {
	ios, in, out, _ := ui.Test()
	// Feed the new password via stdin.
	_, _ = io.WriteString(in, "secret\n")

	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	stubAccount(f)
	stubWithPeers(f)

	var appliedCur, appliedNext string
	opts := &set.Options{
		F:        f,
		NewStdin: true,
		Check: func(_ context.Context) (bool, error) {
			// No existing password.
			return false, nil
		},
		Apply: func(_ context.Context, cur, next, hint string) (bool, error) {
			appliedCur = cur
			appliedNext = next
			return false, nil // hadPrevious = false
		},
	}
	require.NoError(t, set.Run(context.Background(), opts))

	require.Empty(t, appliedCur, "cur must be empty when no existing password")
	require.Equal(t, "secret", appliedNext)

	// Verify emitted JSON row.
	var row map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &row))
	require.Equal(t, "password_set", row["action"])
	// had_previous is false so it should be absent (omitempty).
	_, hasPrev := row["had_previous"]
	require.False(t, hasPrev, "had_previous should be omitted when false")
}

func TestRun_CurrentStdinWithoutExistingPassword_IsUsage(t *testing.T) {
	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	stubAccount(f)
	stubWithPeers(f)

	opts := &set.Options{
		F:            f,
		CurrentStdin: true,
		NewStdin:     true,
		Check: func(_ context.Context) (bool, error) {
			// Account has no password.
			return false, nil
		},
		Apply: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, errors.New("Apply should not be called")
		},
	}
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrUsage)
}

func TestRun_HadPreviousTrue(t *testing.T) {
	ios, in, out, _ := ui.Test()
	// current password on first line, new password on second line.
	_, _ = io.WriteString(in, "oldpwd\nnewpwd\n")

	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	stubAccount(f)
	stubWithPeers(f)

	var appliedCur, appliedNext string
	opts := &set.Options{
		F:            f,
		CurrentStdin: true,
		NewStdin:     true,
		Check: func(_ context.Context) (bool, error) {
			return true, nil // account has an existing password
		},
		Apply: func(_ context.Context, cur, next, hint string) (bool, error) {
			appliedCur = cur
			appliedNext = next
			return true, nil // hadPrevious = true
		},
	}
	require.NoError(t, set.Run(context.Background(), opts))

	require.Equal(t, "oldpwd", appliedCur)
	require.Equal(t, "newpwd", appliedNext)

	// Verify emitted JSON row contains had_previous: true.
	var row map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &row))
	require.Equal(t, "password_set", row["action"])
	require.Equal(t, true, row["had_previous"])
}

func TestRun_HintAndRecoveryEmail_InRow(t *testing.T) {
	ios, in, out, _ := ui.Test()
	_, _ = io.WriteString(in, "mypwd\n")

	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	stubAccount(f)
	stubWithPeers(f)

	opts := &set.Options{
		F:             f,
		NewStdin:      true,
		Hint:          "my hint",
		RecoveryEmail: "me@example.com",
		Check: func(_ context.Context) (bool, error) {
			return false, nil
		},
		Apply: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil
		},
	}
	require.NoError(t, set.Run(context.Background(), opts))

	var row map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &row))
	require.Equal(t, true, row["has_hint"])
	require.Equal(t, true, row["has_recovery_email"])
}

func TestRun_ProbeError_Propagates(t *testing.T) {
	ios, _, _, _ := ui.Test()
	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	stubAccount(f)
	stubWithPeers(f)

	probeErr := errors.New("probe failed")
	opts := &set.Options{
		F: f,
		Check: func(_ context.Context) (bool, error) {
			return false, probeErr
		},
		Apply: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, nil
		},
	}
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, probeErr)
}

func TestRun_ApplyBadPassword_WrapsErrBadPassword(t *testing.T) {
	ios, in, _, _ := ui.Test()
	_, _ = io.WriteString(in, "wrong\n")

	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	stubAccount(f)
	stubWithPeers(f)

	opts := &set.Options{
		F:        f,
		NewStdin: true,
		Check: func(_ context.Context) (bool, error) {
			return false, nil
		},
		Apply: func(_ context.Context, _, _, _ string) (bool, error) {
			return false, fmt.Errorf("%w: bad password", session.ErrBadPassword)
		},
	}
	err := set.Run(context.Background(), opts)
	require.ErrorIs(t, err, session.ErrBadPassword)
}
