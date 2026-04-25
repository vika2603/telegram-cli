package defaults_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/runtime/defaults"
)

func TestNew_wiresAllClosures(t *testing.T) {
	f := defaults.New("test-version")
	require.Equal(t, "test-version", f.AppVersion)
	require.NotNil(t, f.IOStreams)
	require.NotNil(t, f.Prompter)
	require.NotNil(t, f.Config, "Config closure must be wired")
	require.NotNil(t, f.Logger, "Logger closure must be wired")
	require.NotNil(t, f.Account, "Account closure must be wired")
	require.NotNil(t, f.WithClient, "WithClient callback must be wired")
	require.NotNil(t, f.Resolver, "Resolver closure must be wired")
	require.NotNil(t, f.WithPeers, "WithPeers callback must be wired")
}

func TestNew_configClosureUsesInvocationConfigPath(t *testing.T) {
	f := defaults.New("test")
	// Missing file tolerated — returns defaults merged with env.
	f.ConfigPath = t.TempDir() + "/absent.toml"
	cfg, err := f.Config()
	require.NoError(t, err)
	require.NotNil(t, cfg)
}
