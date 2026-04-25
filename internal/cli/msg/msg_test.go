package msg_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/msg"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_HasSubcommands(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := msg.New(f)
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	want := []string{
		"list", "link", "download",
		"send", "edit", "delete", "forward", "react",
		"pin", "unpin",
		"schedule-list", "schedule-cancel",
	}
	for _, n := range want {
		require.True(t, names[n], "expected subcommand %q to be registered", n)
	}
	require.Equal(t, "core", cmd.GroupID)
}
