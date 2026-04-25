package status_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/account/accounttest"
	"github.com/vika2603/telegram-cli/internal/cli/auth/status"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// makeInvocation returns a test Invocation with defaultAccount set (empty = no default).
func makeInvocation(t *testing.T, defaultAccount string) *runtime.Invocation {
	t.Helper()
	f := runtime.NewTestInvocation(t)
	if defaultAccount != "" {
		da := defaultAccount
		cfg := config.Defaults()
		cfg.DefaultAccount = &da
		f.Config = func() (*config.Config, error) { return &cfg, nil }
	}
	return f
}

// ---- Tests ----------------------------------------------------------------

func TestRun_DefaultSlot_WhenNameOmitted(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")

	workAcct := accounttest.SeedAccount(t, "", "work", account.StateAUTHED)
	f.Account = func(n string) (*account.Account, error) {
		if n == "" || n == "work" {
			return workAcct, nil
		}
		return nil, fmt.Errorf("account not found: %s", n)
	}

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	opts := &status.Options{
		F:    f,
		Name: "", // positional arg omitted — falls back to default "work"
	}
	require.NoError(t, status.Run(context.Background(), opts))
	require.Contains(t, out.String(), "name: work")
	require.Contains(t, out.String(), "default: true")
}

func TestRun_PositionalSlot_Overrides(t *testing.T) {
	accounttest.TempConfigRoot(t)
	// default is "work", but positional arg selects "other"
	f := makeInvocation(t, "work")

	workAcct := accounttest.SeedAccount(t, "", "work", account.StateAUTHED)
	otherAcct := accounttest.SeedAccount(t, "", "other", account.StateAUTHED)
	f.Account = func(n string) (*account.Account, error) {
		switch n {
		case "work", "":
			return workAcct, nil
		case "other":
			return otherAcct, nil
		default:
			return nil, fmt.Errorf("account not found: %s", n)
		}
	}

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	opts := &status.Options{
		F:    f,
		Name: "other",
	}
	require.NoError(t, status.Run(context.Background(), opts))
	require.Contains(t, out.String(), "name: other")
	require.Contains(t, out.String(), "default: false")
}

func TestRun_AUTHED_NoProbe(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateAUTHED)
	f.Account = func(n string) (*account.Account, error) {
		if n == "" || n == "work" {
			return acct, nil
		}
		return nil, fmt.Errorf("account not found: %s", n)
	}

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	opts := &status.Options{
		F:     f,
		Name:  "work",
		Probe: false,
	}
	require.NoError(t, status.Run(context.Background(), opts))
	require.Contains(t, out.String(), "state: AUTHED")
	// probed line should not appear when --probe not set
	require.NotContains(t, out.String(), "probed:")
}

func TestRun_SessionModified_Present(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateAUTHED)
	f.Account = func(n string) (*account.Account, error) {
		if n == "" || n == "work" {
			return acct, nil
		}
		return nil, fmt.Errorf("account not found: %s", n)
	}

	// Pre-create session.bin at a known time.
	sessPath := account.SessionFile("work")
	require.NoError(t, os.WriteFile(sessPath, []byte("x"), 0600))
	knownTime := time.Date(2024, 11, 2, 15, 32, 10, 0, time.UTC)
	require.NoError(t, os.Chtimes(sessPath, knownTime, knownTime))

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	opts := &status.Options{
		F:    f,
		Name: "work",
	}
	require.NoError(t, status.Run(context.Background(), opts))
	require.Contains(t, out.String(), "session_modified: 2024-11-02T15:32:10Z")
}

func TestRun_SessionModified_Absent(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	f.Account = func(n string) (*account.Account, error) {
		if n == "" || n == "work" {
			return acct, nil
		}
		return nil, fmt.Errorf("account not found: %s", n)
	}

	// Make sure session.bin does not exist.
	_ = os.Remove(account.SessionFile("work"))

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	opts := &status.Options{
		F:    f,
		Name: "work",
	}
	require.NoError(t, status.Run(context.Background(), opts))
	// Human output shows "(none)" when absent.
	require.Contains(t, out.String(), "session_modified: (none)")
}

func TestRun_Probe_AUTHED_Success(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateAUTHED)
	f.Account = func(n string) (*account.Account, error) {
		if n == "" || n == "work" {
			return acct, nil
		}
		return nil, fmt.Errorf("account not found: %s", n)
	}
	f.WithClient = func(ctx context.Context, _ *account.Account, _ session.Options,
		fn func(context.Context, session.Client) error) error {
		return fn(ctx, &session.FakeClient{})
	}

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	probeCalled := false
	opts := &status.Options{
		F:     f,
		Name:  "work",
		Probe: true,
		DoProbe: func(_ context.Context, _ *tg.Client) error {
			probeCalled = true
			return nil
		},
	}
	require.NoError(t, status.Run(context.Background(), opts))
	require.True(t, probeCalled, "DoProbe must have been called")
	require.Contains(t, out.String(), "probed: true")
	require.Contains(t, out.String(), "probe_ok: true")
}

func TestRun_Probe_AUTHED_Failure(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateAUTHED)
	f.Account = func(n string) (*account.Account, error) {
		if n == "" || n == "work" {
			return acct, nil
		}
		return nil, fmt.Errorf("account not found: %s", n)
	}
	f.WithClient = func(ctx context.Context, _ *account.Account, _ session.Options,
		fn func(context.Context, session.Client) error) error {
		return fn(ctx, &session.FakeClient{})
	}

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	opts := &status.Options{
		F:     f,
		Name:  "work",
		Probe: true,
		DoProbe: func(_ context.Context, _ *tg.Client) error {
			return errors.New("probe failed")
		},
	}
	// Run must NOT propagate probe error.
	require.NoError(t, status.Run(context.Background(), opts))
	require.Contains(t, out.String(), "probed: true")
	require.Contains(t, out.String(), "probe_ok: false")
}

func TestRun_Probe_NEW_SkipsProbe(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	f.Account = func(n string) (*account.Account, error) {
		if n == "" || n == "work" {
			return acct, nil
		}
		return nil, fmt.Errorf("account not found: %s", n)
	}

	ios, _, out, errOut := ui.Test()
	f.IOStreams = ios

	opts := &status.Options{
		F:     f,
		Name:  "work",
		Probe: true,
		DoProbe: func(_ context.Context, _ *tg.Client) error {
			panic("DoProbe must not be called for non-AUTHED slot")
		},
	}
	require.NoError(t, status.Run(context.Background(), opts))
	require.Contains(t, out.String(), "probed: true")
	require.Contains(t, out.String(), "probe_ok: false")
	require.Contains(t, errOut.String(), "skipping probe")
}

func TestRun_Probe_EXPIRED_SkipsProbe(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateEXPIRED)
	f.Account = func(n string) (*account.Account, error) {
		if n == "" || n == "work" {
			return acct, nil
		}
		return nil, fmt.Errorf("account not found: %s", n)
	}

	ios, _, out, errOut := ui.Test()
	f.IOStreams = ios

	opts := &status.Options{
		F:     f,
		Name:  "work",
		Probe: true,
		DoProbe: func(_ context.Context, _ *tg.Client) error {
			panic("DoProbe must not be called for non-AUTHED slot")
		},
	}
	require.NoError(t, status.Run(context.Background(), opts))
	require.Contains(t, out.String(), "probed: true")
	require.Contains(t, out.String(), "probe_ok: false")
	require.Contains(t, errOut.String(), "skipping probe")
}

func TestRun_APIID_Populated(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")

	// Seed with a specific APIID.
	meta := account.Meta{
		Name:  "work",
		State: account.StateAUTHED,
		APIID: 12345,
	}
	require.NoError(t, account.AddAccount(meta))
	require.NoError(t, account.SetState("work", account.StateAUTHED))
	acct, err := account.LoadAccount("work")
	require.NoError(t, err)
	f.Account = func(n string) (*account.Account, error) {
		if n == "" || n == "work" {
			return acct, nil
		}
		return nil, fmt.Errorf("account not found: %s", n)
	}

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	opts := &status.Options{
		F:    f,
		Name: "work",
	}
	require.NoError(t, status.Run(context.Background(), opts))
	require.Contains(t, out.String(), "api_id: 12345")
}

func TestRun_UnknownSlot_IsError(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "")
	f.Account = func(n string) (*account.Account, error) {
		return nil, fmt.Errorf("account not found: %s", n)
	}

	opts := &status.Options{
		F:    f,
		Name: "nope",
	}
	err := status.Run(context.Background(), opts)
	require.Error(t, err)
	require.Contains(t, err.Error(), "nope")
}
