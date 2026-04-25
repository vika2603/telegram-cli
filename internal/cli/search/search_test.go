package search_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/cli/search"
	"github.com/vika2603/telegram-cli/internal/command"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestNew_HasSubcommands(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := search.New(f)
	names := map[string]bool{}
	for _, c := range cmd.Commands() {
		names[c.Name()] = true
	}
	require.True(t, names["msg"])
	require.True(t, names["chat"])
	require.Equal(t, "core", cmd.GroupID)
}

func TestNew_BareSearchDefaultsToMsg(t *testing.T) {
	// Per spec §6.7: "Bare `tg search <q>` (no sub-verb) defaults to
	// `search msg <q>`." We verify by checking that the root RunE is
	// registered.
	f := runtime.NewTestInvocation(t)
	cmd := search.New(f)
	require.NotNil(t, cmd.RunE, "bare 'tg search <q>' must have RunE that shims to msg subcommand")
}

func TestNew_BareSearchPassesMsgFlags(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	cmd := search.New(f)
	cmd.SetArgs([]string{"hello", "--limit", "1"})
	err := cmd.Execute()
	require.ErrorIs(t, err, command.ErrPrecondition)
	require.NotContains(t, err.Error(), "unknown flag")
}
