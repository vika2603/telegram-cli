package member_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/chat/member"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_HasListSubcommand(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := member.New(f)
	require.Equal(t, "member", cmd.Name())
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	require.True(t, names["list"], "expected subcommand \"list\" to be registered")
}
