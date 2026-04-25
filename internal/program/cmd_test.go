package program

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/root"
	"github.com/vika2603/telegram-cli/internal/runtime/defaults"
)

func TestResolveErrorMode_defaultsToHuman(t *testing.T) {
	f := defaults.New("test")
	rootCmd := root.New(f)
	require.Equal(t, "human", resolveErrorMode(rootCmd, f))
}

func TestResolveErrorMode_outputFlagWins(t *testing.T) {
	f := defaults.New("test")
	rootCmd := root.New(f)
	require.NoError(t, rootCmd.PersistentFlags().Set("output", "json"))
	require.Equal(t, "json", resolveErrorMode(rootCmd, f))
}

// TestResolveErrorMode_jsonFlagWins exercises the --json-on-leaf path:
// resolveErrorMode is called with the executed leaf cmd, so when a leaf
// like `auth list` has its own --json flag flipped, error rendering
// flips to JSON regardless of the root --output flag.
func TestResolveErrorMode_jsonFlagWins(t *testing.T) {
	f := defaults.New("test")
	rootCmd := root.New(f)
	leaf, _, err := rootCmd.Find([]string{"auth", "list"})
	require.NoError(t, err)
	require.NotNil(t, leaf)
	require.NoError(t, leaf.Flags().Set("json", "name"))
	require.Equal(t, "json", resolveErrorMode(leaf, f))
}
