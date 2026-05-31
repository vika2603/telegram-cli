package chat_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/chat"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_HasSubcommands(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := chat.New(f)
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	want := []string{"list", "info", "member", "topic", "mark-read", "join", "leave", "mute", "unmute", "archive", "unarchive", "pin", "unpin"}
	for _, n := range want {
		require.True(t, names[n], "expected subcommand %q to be registered", n)
	}
	require.Equal(t, "core", cmd.GroupID)
}
