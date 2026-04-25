package switchcmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/account/accounttest"
	switchcmd "github.com/vika2603/telegram-cli/internal/cli/auth/switch"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/config"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

// TestRun_InvalidName verifies that a name containing invalid characters
// returns ErrUsage without touching disk.
func TestRun_InvalidName(t *testing.T) {
	_ = accounttest.TempConfigRoot(t)
	f := runtime.NewTestInvocation(t)
	opts := &switchcmd.Options{
		Name: "foo/bar",
		F:    f,
	}
	err := switchcmd.Run(opts)
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
}

// TestRun_NonExistent verifies that a valid but non-existent account name
// returns ErrUsage with a message mentioning "does not exist".
func TestRun_NonExistent(t *testing.T) {
	_ = accounttest.TempConfigRoot(t)
	f := runtime.NewTestInvocation(t)
	opts := &switchcmd.Options{
		Name: "ghostaccount",
		F:    f,
	}
	err := switchcmd.Run(opts)
	require.Error(t, err)
	require.ErrorIs(t, err, command.ErrUsage)
	require.Contains(t, err.Error(), "does not exist")
}

// TestRun_WritesDefault is the happy-path test: pre-create an account slot,
// run switch, re-read the config file and assert default_account is set.
func TestRun_WritesDefault(t *testing.T) {
	root := accounttest.TempConfigRoot(t)
	accounttest.SeedAccount(t, root, "personal", account.StateAUTHED)

	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	f := runtime.NewTestInvocation(t)
	f.ConfigPath = cfgPath

	opts := &switchcmd.Options{
		Name: "personal",
		F:    f,
	}
	require.NoError(t, switchcmd.Run(opts))

	raw, err := config.ReadRawAt(cfgPath)
	require.NoError(t, err)
	require.Equal(t, "personal", raw["default_account"])
}

// TestRun_RespectsConfigPathFlag is the regression-prevention test for the
// documented behavior change vs account use: when f.ConfigPath is set, the
// default must be written there, NOT to the standard account.ConfigFile().
func TestRun_RespectsConfigPathFlag(t *testing.T) {
	root := accounttest.TempConfigRoot(t)
	accounttest.SeedAccount(t, root, "work", account.StateAUTHED)

	// alt.toml is the explicit config path the caller wants to use.
	altPath := filepath.Join(t.TempDir(), "alt.toml")
	// defaultPath is where account.ConfigFile() resolves to under the temp root.
	defaultPath := account.ConfigFile()

	f := runtime.NewTestInvocation(t)
	f.ConfigPath = altPath

	opts := &switchcmd.Options{
		Name: "work",
		F:    f,
	}
	require.NoError(t, switchcmd.Run(opts))

	// alt.toml must be written.
	raw, err := config.ReadRawAt(altPath)
	require.NoError(t, err)
	require.Equal(t, "work", raw["default_account"])

	// The standard location must NOT have been touched.
	_, statErr := os.Stat(defaultPath)
	require.True(t, os.IsNotExist(statErr), "default config path should not have been created")
}
