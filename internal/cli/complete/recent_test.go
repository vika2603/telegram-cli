package complete_test

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/vika2603/telegram-cli/internal/account"
	"github.com/vika2603/telegram-cli/internal/cli/complete"
	"github.com/vika2603/telegram-cli/internal/runtime"
)

func TestPeerRefs_UsesRecentPeerDescriptions(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	require.NoError(t, os.MkdirAll(account.AccountDir("work"), 0o700))
	db, err := account.OpenPeersDB("work")
	require.NoError(t, err)
	store := account.NewPeerStore(db)
	require.NoError(t, store.RecordRecentPeer(account.RecentPeer{
		Ref:      "@alice",
		ID:       42,
		Kind:     "user",
		Title:    "Alice Chen",
		Username: "alice",
	}))
	require.NoError(t, db.Close())

	f := runtime.NewTestInvocation(t)
	f.AccountName = "work"
	got, directive := complete.PeerRefs(f)(&cobra.Command{}, nil, "ali")
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	require.Equal(t, []string{"@alice\tAlice Chen @alice user id:42"}, got)
}

func TestPeerRefs_OnlyCompletesPeerArgument(t *testing.T) {
	f := runtime.NewTestInvocation(t)
	got, directive := complete.PeerRefs(f)(&cobra.Command{}, []string{"@alice"}, "")
	require.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	require.Empty(t, got)
}
