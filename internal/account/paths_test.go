package account

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoot_respectsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/custom/xdg")
	require.Equal(t, "/custom/xdg/tg", Root())
}

func TestRoot_defaultHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	require.Equal(t, filepath.Join(home, ".config", "tg"), Root())
}

func TestAccountDir(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/x")
	require.Equal(t, "/x/tg/accounts/alice", AccountDir("alice"))
}
