package complete_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestPeerRefs_UsesRecentPeerDescriptions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, account.AddAccount(account.Meta{Name: "work", State: account.StateNEW}))
	store, err := account.OpenRecentStore("work")
	require.NoError(t, err)
	require.NoError(t, store.RecordRecentPeer(account.RecentPeer{
		Ref: "@alice", Title: "Alice", Username: "alice", Kind: "user", ID: 42,
	}))

	cmd := &cobra.Command{Use: "x"}
	got, directive := complete.PeerRefs(&runtime.Invocation{AccountName: "work"})(cmd, nil, "ali")
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	require.Equal(t, []string{"@alice\tAlice @alice user id:42"}, got)
}
