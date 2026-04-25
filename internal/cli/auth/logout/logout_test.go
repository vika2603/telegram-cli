package logout_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/account/accounttest"
	"github.com/vika2603/telegram-cli/internal/cli/auth/logout"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/runtime"
	"github.com/vika2603/telegram-cli/internal/telegram/session"
	"github.com/vika2603/telegram-cli/internal/ui"
)

// capturingPrompter records the prompt message passed to Confirm.
type capturingPrompter struct {
	prompt  string
	confirm bool
}

func (p *capturingPrompter) Confirm(prompt string, _ bool) (bool, error) {
	p.prompt = prompt
	return p.confirm, nil
}

func (p *capturingPrompter) Input(_, _ string) (string, error) { return "", nil }
func (p *capturingPrompter) Password(_ string) (string, error) { return "", nil }

// makeInvocation returns a test Invocation wired with a Config closure that returns
// a config with the given defaultAccount (empty string = nil pointer).
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

// mustReadTOML parses a TOML file into map[string]any.
func mustReadTOML(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, toml.Unmarshal(data, &raw))
	return raw
}

// ---- Tests ----------------------------------------------------------------

func TestRun_NoDo_Precondition(t *testing.T) {
	f := makeInvocation(t, "")
	opts := &logout.Options{
		F:  f,
		Do: nil,
	}
	err := logout.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrPrecondition)
}

func TestRun_NameFromDefault_WhenArgOmitted(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	f.Account = func(n string) (*account.Account, error) {
		// empty name → default slot
		if n == "" || n == "work" {
			return acct, nil
		}
		return nil, fmt.Errorf("unexpected account %q", n)
	}
	f.WithClient = func(_ context.Context, _ *account.Account, _ session.Options,
		_ func(context.Context, session.Client) error) error {
		panic("WithClient must not be called for NEW slot")
	}

	opts := &logout.Options{
		F:    f,
		Yes:  true,
		Name: "", // positional arg omitted
		Do:   func(_ context.Context, _ logout.DoArgs) error { return nil },
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err)

	// Verify the slot resolved to "work": meta name is "work".
	m, merr := account.ReadMeta("work")
	require.NoError(t, merr)
	require.Equal(t, "work", m.Name)
}

func TestRun_NameFromArg_Overrides(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	workAcct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	otherAcct := accounttest.SeedAccount(t, "", "other", account.StateNEW)
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
	f.WithClient = func(_ context.Context, _ *account.Account, _ session.Options,
		_ func(context.Context, session.Client) error) error {
		panic("WithClient must not be called for NEW slot")
	}

	opts := &logout.Options{
		F:    f,
		Yes:  true,
		Name: "other",
		Do:   func(_ context.Context, _ logout.DoArgs) error { return nil },
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err)

	// "other" should have State=NEW (WriteMeta was called with NewState).
	m, merr := account.ReadMeta("other")
	require.NoError(t, merr)
	require.Equal(t, "other", m.Name)
}

func TestRun_Logout_NotConfirmed(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	f.Account = func(_ string) (*account.Account, error) { return acct, nil }
	f.Prompter = &ui.StubPrompter{Answers: []any{false}}

	opts := &logout.Options{
		F:    f,
		Yes:  false,
		Name: "work",
		Do:   func(_ context.Context, _ logout.DoArgs) error { return nil },
	}
	err := logout.Run(context.Background(), opts)
	require.ErrorIs(t, err, command.ErrNotConfirmed)
}

func TestRun_Logout_Success_AUTHED(t *testing.T) {
	accounttest.TempConfigRoot(t)
	ios, _, out, _ := ui.Test()
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateAUTHED)
	f.IOStreams = ios
	f.Account = func(_ string) (*account.Account, error) { return acct, nil }
	f.WithClient = func(ctx context.Context, _ *account.Account, _ session.Options,
		fn func(context.Context, session.Client) error) error {
		return fn(ctx, &session.FakeClient{})
	}

	// Write a fake session file to verify it gets deleted inside Do.
	require.NoError(t, os.WriteFile(account.SessionFile("work"), []byte("data"), 0600))

	var capturedName string
	opts := &logout.Options{
		F:    f,
		Yes:  true,
		Name: "work",
		Do: func(_ context.Context, a logout.DoArgs) error {
			capturedName = a.AcctName
			// Simulate production Do: delete session + flip state.
			_ = account.DeleteSession(a.AcctName)
			m, err := account.ReadMeta(a.AcctName)
			if err != nil {
				return err
			}
			m.State = account.StateNEW
			return account.WriteMeta(m)
		},
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, "work", capturedName)

	// Session file should be gone.
	_, serr := os.Stat(account.SessionFile("work"))
	require.True(t, os.IsNotExist(serr), "session.bin should have been deleted")

	// Account directory still exists (no --purge).
	require.DirExists(t, account.AccountDir("work"))

	// Meta.State is NEW.
	m, merr := account.ReadMeta("work")
	require.NoError(t, merr)
	require.Equal(t, account.StateNEW, m.State)

	// Human output.
	require.Contains(t, out.String(), "logged out work")
	require.NotContains(t, out.String(), "(purged)")
}

func TestRun_Logout_AlreadyLoggedOut_ServerError_IsTolerated(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateAUTHED)
	f.Account = func(_ string) (*account.Account, error) { return acct, nil }
	f.WithClient = func(ctx context.Context, _ *account.Account, _ session.Options,
		fn func(context.Context, session.Client) error) error {
		return fn(ctx, &session.FakeClient{})
	}

	opts := &logout.Options{
		F:    f,
		Yes:  true,
		Name: "work",
		Do: func(_ context.Context, _ logout.DoArgs) error {
			return errors.New("rpc: AUTH_KEY_UNREGISTERED")
		},
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err, "AUTH_KEY_UNREGISTERED must be swallowed")
}

func TestRun_Logout_NEW_SkipsClientAndServer(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	f.Account = func(_ string) (*account.Account, error) { return acct, nil }
	f.WithClient = func(_ context.Context, _ *account.Account, _ session.Options,
		_ func(context.Context, session.Client) error) error {
		panic("WithClient must not be called for NEW slot")
	}

	// Write a fake session file to confirm DeleteSession is called.
	require.NoError(t, os.WriteFile(account.SessionFile("work"), []byte("x"), 0600))

	opts := &logout.Options{
		F:    f,
		Yes:  true,
		Name: "work",
		Do:   func(_ context.Context, _ logout.DoArgs) error { return nil },
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err)

	// Session file deleted by the non-client path.
	_, serr := os.Stat(account.SessionFile("work"))
	require.True(t, os.IsNotExist(serr))

	// Meta state is still NEW (WriteMeta wrote it back as NEW).
	m, merr := account.ReadMeta("work")
	require.NoError(t, merr)
	require.Equal(t, account.StateNEW, m.State)
}

func TestRun_Logout_EXPIRED_SkipsClient(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateEXPIRED)
	f.Account = func(_ string) (*account.Account, error) { return acct, nil }
	f.WithClient = func(_ context.Context, _ *account.Account, _ session.Options,
		_ func(context.Context, session.Client) error) error {
		panic("WithClient must not be called for EXPIRED slot")
	}

	opts := &logout.Options{
		F:    f,
		Yes:  true,
		Name: "work",
		Do:   func(_ context.Context, _ logout.DoArgs) error { return nil },
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err)

	// Meta state flipped to NEW.
	m, merr := account.ReadMeta("work")
	require.NoError(t, merr)
	require.Equal(t, account.StateNEW, m.State)
}

func TestRun_Purge_Success_NotDefault(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	workAcct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	otherAcct := accounttest.SeedAccount(t, "", "other", account.StateNEW)
	f.Account = func(n string) (*account.Account, error) {
		if n == "other" {
			return otherAcct, nil
		}
		return workAcct, nil
	}
	f.WithClient = func(_ context.Context, _ *account.Account, _ session.Options,
		_ func(context.Context, session.Client) error) error {
		panic("WithClient must not be called for NEW slot")
	}

	cfgPath := account.ConfigFile()
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0700))
	require.NoError(t, config.SetDefaultAccount(cfgPath, "work"))
	f.ConfigPath = cfgPath

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	opts := &logout.Options{
		F:     f,
		Yes:   true,
		Purge: true,
		Name:  "other",
		Do:    func(_ context.Context, _ logout.DoArgs) error { return nil },
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err)

	// AccountDir("other") is gone.
	require.NoDirExists(t, account.AccountDir("other"))

	// config.toml default_account still "work".
	raw := mustReadTOML(t, cfgPath)
	require.Equal(t, "work", raw["default_account"])

	require.Contains(t, out.String(), "(purged)")
	require.NotContains(t, out.String(), "[default cleared]")
}

func TestRun_Purge_Success_WasDefault(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	f.Account = func(_ string) (*account.Account, error) { return acct, nil }
	f.WithClient = func(_ context.Context, _ *account.Account, _ session.Options,
		_ func(context.Context, session.Client) error) error {
		panic("WithClient must not be called for NEW slot")
	}

	cfgPath := account.ConfigFile()
	require.NoError(t, os.MkdirAll(filepath.Dir(cfgPath), 0700))
	require.NoError(t, config.SetDefaultAccount(cfgPath, "work"))
	f.ConfigPath = cfgPath

	ios, _, out, _ := ui.Test()
	f.IOStreams = ios

	opts := &logout.Options{
		F:     f,
		Yes:   true,
		Purge: true,
		Name:  "work",
		Do:    func(_ context.Context, _ logout.DoArgs) error { return nil },
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err)

	// AccountDir gone.
	require.NoDirExists(t, account.AccountDir("work"))

	// config.toml has no default_account.
	raw := mustReadTOML(t, cfgPath)
	_, hasKey := raw["default_account"]
	require.False(t, hasKey, "default_account should be absent from config.toml")

	require.Contains(t, out.String(), "(purged)")
	require.Contains(t, out.String(), "[default cleared]")
}

func TestRun_Purge_UnsetDefaultFails_PurgeStillSucceeds(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	f.Account = func(_ string) (*account.Account, error) { return acct, nil }
	f.WithClient = func(_ context.Context, _ *account.Account, _ session.Options,
		_ func(context.Context, session.Client) error) error {
		panic("WithClient must not be called for NEW slot")
	}

	// Write a config file in a separate temp dir that we make read-only.
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.toml")
	require.NoError(t, config.SetDefaultAccount(cfgPath, "work"))
	// Make the file read-only and the directory non-writable.
	require.NoError(t, os.Chmod(cfgPath, 0400))
	require.NoError(t, os.Chmod(cfgDir, 0500))
	// Restore permissions so t.TempDir cleanup can remove them.
	t.Cleanup(func() {
		_ = os.Chmod(cfgDir, 0700)
		_ = os.Chmod(cfgPath, 0600)
	})

	f.ConfigPath = cfgPath

	ios, _, _, errOut := ui.Test()
	f.IOStreams = ios

	opts := &logout.Options{
		F:     f,
		Yes:   true,
		Purge: true,
		Name:  "work",
		Do:    func(_ context.Context, _ logout.DoArgs) error { return nil },
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err, "purge must succeed even when UnsetDefaultAccount fails")

	// AccountDir is gone (purge succeeded).
	require.NoDirExists(t, account.AccountDir("work"))

	// stderr warning was emitted.
	require.Contains(t, errOut.String(), "could not clear default_account")
	require.Contains(t, errOut.String(), "manually")
}

func TestRun_Purge_PromptMessage_MentionsAutoClear(t *testing.T) {
	accounttest.TempConfigRoot(t)
	f := makeInvocation(t, "work")
	acct := accounttest.SeedAccount(t, "", "work", account.StateNEW)
	f.Account = func(_ string) (*account.Account, error) { return acct, nil }
	f.WithClient = func(_ context.Context, _ *account.Account, _ session.Options,
		_ func(context.Context, session.Client) error) error {
		panic("WithClient must not be called for NEW slot")
	}

	cp := &capturingPrompter{confirm: true}
	f.Prompter = cp

	opts := &logout.Options{
		F:     f,
		Yes:   false, // must go through Confirm
		Purge: true,
		Name:  "work",
		Do:    func(_ context.Context, _ logout.DoArgs) error { return nil },
	}
	err := logout.Run(context.Background(), opts)
	require.NoError(t, err)
	require.Contains(t, cp.prompt, "default_account pointer will be cleared")
}
