package contact_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/contact"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_HasGroupID(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := contact.New(f)
	require.Equal(t, "core", cmd.GroupID)
	require.Equal(t, "contact", cmd.Name())
}

func TestNew_HasSubcommands(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := contact.New(f)
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	want := []string{"list", "add", "delete", "block", "unblock"}
	for _, n := range want {
		require.True(t, names[n], "expected subcommand %q to be registered", n)
	}
}
