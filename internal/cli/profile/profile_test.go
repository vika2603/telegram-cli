package profile_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/profile"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_HasGroupID(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := profile.New(f)
	require.Equal(t, "core", cmd.GroupID)
	require.Equal(t, "profile", cmd.Name())
}

func TestNew_HasSubcommands(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := profile.New(f)
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	want := []string{
		"set-name", "set-username", "set-bio",
		"set-photo", "delete-photo", "set-status",
	}
	for _, n := range want {
		require.True(t, names[n], "expected subcommand %q to be registered", n)
	}
}
