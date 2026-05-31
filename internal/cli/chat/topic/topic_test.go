package topic_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/chat/topic"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_HasSubcommands(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := topic.New(f)
	require.Equal(t, "topic", cmd.Name())
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"list", "create", "edit", "delete", "pin", "unpin", "info", "mute", "unmute", "read"} {
		require.True(t, names[want], "expected subcommand %q to be registered", want)
	}
}
