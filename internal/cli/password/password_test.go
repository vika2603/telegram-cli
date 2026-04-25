package password_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/password"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_GroupProperties(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := password.New(f)
	require.Equal(t, "password", cmd.Name())
	require.Equal(t, "setup", cmd.GroupID)
}

func TestNew_SetSubcommandRegistered(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := password.New(f)
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	require.True(t, names["set"], "expected subcommand 'set' to be registered")
}
