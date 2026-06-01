package photo_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/chat/photo"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_HasSubcommands(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := photo.New(f)
	require.Equal(t, "photo", cmd.Name())
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"set", "clear"} {
		require.True(t, names[want], "expected subcommand %q to be registered", want)
	}
}
