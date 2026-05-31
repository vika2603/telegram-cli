package root_test

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/cli/root"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_HasExpectedGroups(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := root.New(f)
	require.Equal(t, "tg", cmd.Use)
	require.True(t, cmd.SilenceErrors)
	require.True(t, cmd.SilenceUsage)

	groups := cmd.Groups()
	ids := map[string]bool{}
	for _, g := range groups {
		ids[g.ID] = true
	}
	require.True(t, ids["core"])
	require.True(t, ids["setup"])
	require.True(t, ids["frequent"])
}

func TestNew_HelpAcceptsNoArgs(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := root.New(f)
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	require.NoError(t, err)
}

func TestNew_HasTopLevels(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := root.New(f)
	want := []string{
		"login", "logout", "send", "reply", "inbox", "read", "digest", "resolve",
		"chat", "channel", "msg", "search", "me", "contact", "profile",
		"auth", "sessions", "password", "config", "completion",
	}
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	for _, n := range want {
		require.True(t, names[n], "missing top-level %q", n)
	}
}

// TestRootPreRun_AccountFromArg_SkipsPreload verifies that when a command sets
// Meta.AccountFromArg = true, the root PersistentPreRunE returns nil without
// ever calling f.Account (i.e., it does NOT preload the default slot).
func TestRootPreRun_AccountFromArg_SkipsPreload(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	// Override Account to panic — any call means the test fails.
	f.Account = func(_ string) (*account.Account, error) {
		panic("Account must not be called when AccountFromArg is true")
	}

	// Build a minimal root tree and attach a child command that has AccountFromArg.
	rootCmd := root.New(f)
	child := &cobra.Command{
		Use:  "slot",
		RunE: func(_ *cobra.Command, _ []string) error { return nil },
	}
	command.SetMeta(child, command.Meta{AccountFromArg: true})
	rootCmd.AddCommand(child)

	rootCmd.SetArgs([]string{"slot"})
	err := rootCmd.ExecuteContext(context.Background())
	require.NoError(t, err)
}
