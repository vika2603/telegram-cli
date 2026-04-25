package disable_test

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
	"github.com/vika2603/telegram-cli/internal/cli/password/disable"
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
	var captured *disable.Options
	f := runtime.NewTestInvocation(t)
	cmd := disable.New(f, func(o *disable.Options) error { captured = o; return nil })
	cmd.SetArgs([]string{
		"--current-stdin",
		"--yes",
	})
	require.NoError(t, cmd.Execute())
	require.True(t, captured.CurrentStdin)
	require.True(t, captured.Yes)
}

func TestRun_NilProbeReturnsPrecondition(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)

	opts := &disable.Options{
		F:     f,
		Check: nil,
		Apply: func(_ context.Context, _ string) error { return nil },
	}
	err := disable.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_NilApplyReturnsPrecondition(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)

	opts := &disable.Options{
		F:     f,
		Check: func(_ context.Context) (bool, error) { return true, nil },
		Apply: nil,
	}
	err := disable.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_NoPasswordIsPrecondition(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	stubAccount(f)
	stubWithPeers(f)

	opts := &disable.Options{
		F: f,
		Check: func(_ context.Context) (bool, error) {
			return false, nil // no password set
		},
		Apply: func(_ context.Context, _ string) error {
			return errors.New("Apply should not be called")
		},
	}
	err := disable.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
	require.Contains(t, err.Error(), "nothing to disable")
}

func TestRun_DeclinedPrompt(t *testing.T) {
	ios, in, _, _ := ui.Test()
	// Feed the current password via stdin.
	_, _ = io.WriteString(in, "mypwd\n")

	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	f.Prompter = &ui.StubPrompter{Answers: []any{false}}
	stubAccount(f)
	stubWithPeers(f)

	opts := &disable.Options{
		F:            f,
		CurrentStdin: true,
		Check: func(_ context.Context) (bool, error) {
			return true, nil
		},
		Apply: func(_ context.Context, _ string) error {
			return errors.New("Apply should not be called")
		},
	}
	err := disable.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestRun_YesBypassesPrompt(t *testing.T) {
	ios, in, out, _ := ui.Test()
	_, _ = io.WriteString(in, "mypwd\n")

	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	stubAccount(f)
	stubWithPeers(f)

	var appliedCur string
	opts := &disable.Options{
		F:            f,
		CurrentStdin: true,
		Yes:          true,
		Check: func(_ context.Context) (bool, error) {
			return true, nil
		},
		Apply: func(_ context.Context, cur string) error {
			appliedCur = cur
			return nil
		},
	}
	require.NoError(t, disable.Run(context.Background(), opts))
	require.Equal(t, "mypwd", appliedCur)

	// Verify emitted JSON row.
	var row map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &row))
	require.Equal(t, "password_disable", row["action"])
}

func TestRun_StdinReadsPasswordThenApply(t *testing.T) {
	ios, in, out, _ := ui.Test()
	_, _ = io.WriteString(in, "secret\n")

	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	stubAccount(f)
	stubWithPeers(f)

	var appliedCur string
	opts := &disable.Options{
		F:            f,
		CurrentStdin: true,
		Yes:          true,
		Check: func(_ context.Context) (bool, error) {
			return true, nil
		},
		Apply: func(_ context.Context, cur string) error {
			appliedCur = cur
			return nil
		},
	}
	require.NoError(t, disable.Run(context.Background(), opts))
	require.Equal(t, "secret", appliedCur)

	var row map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &row))
	require.Equal(t, "password_disable", row["action"])
}

func TestRun_BadPasswordError(t *testing.T) {
	ios, in, _, _ := ui.Test()
	_, _ = io.WriteString(in, "wrongpwd\n")

	f := runtime.NewTestInvocation(t)
	f.IOStreams = ios
	stubAccount(f)
	stubWithPeers(f)

	opts := &disable.Options{
		F:            f,
		CurrentStdin: true,
		Yes:          true,
		Check: func(_ context.Context) (bool, error) {
			return true, nil
		},
		Apply: func(_ context.Context, _ string) error {
			return fmt.Errorf("%w: PASSWORD_HASH_INVALID", session.ErrBadPassword)
		},
	}
	err := disable.Run(context.Background(), opts)
	require.ErrorIs(t, err, session.ErrBadPassword)
}
