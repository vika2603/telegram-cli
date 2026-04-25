package session_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/session"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_GroupProperties(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := session.New(f)
	require.Equal(t, "sessions", cmd.Name())
	require.Equal(t, "setup", cmd.GroupID)
}

func TestNew_HasListSubcommand(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := session.New(f)
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	require.True(t, names["list"], "expected subcommand \"list\" to be registered")
	require.True(t, names["revoke"], "expected subcommand \"revoke\" to be registered")
}
