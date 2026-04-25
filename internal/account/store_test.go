package account

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTempStore(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestStore_AddAndList(t *testing.T) {
	setupTempStore(t)
	require.NoError(t, AddAccount(Meta{Name: "alice", State: StateNEW, APIID: 1, APIHash: "h"}))
	require.NoError(t, AddAccount(Meta{Name: "bob", State: StateNEW, APIID: 1, APIHash: "h"}))
	names, err := ListAccounts()
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"alice", "bob"}, names)
}

func TestStore_Add_rejectsDuplicate(t *testing.T) {
	setupTempStore(t)
	require.NoError(t, AddAccount(Meta{Name: "alice", State: StateNEW, APIID: 1, APIHash: "h"}))
	err := AddAccount(Meta{Name: "alice", State: StateNEW, APIID: 1, APIHash: "h"})
	require.Error(t, err)
}

func TestStore_Resolve_explicitOverridesDefault(t *testing.T) {
	setupTempStore(t)
	require.NoError(t, AddAccount(Meta{Name: "alice", State: StateNEW, APIID: 1, APIHash: "h"}))
	require.NoError(t, AddAccount(Meta{Name: "bob", State: StateNEW, APIID: 1, APIHash: "h"}))
	name, err := ResolveAccount("bob", "alice")
	require.NoError(t, err)
	require.Equal(t, "bob", name)
	name, err = ResolveAccount("", "alice")
	require.NoError(t, err)
	require.Equal(t, "alice", name)
}

func TestStore_Resolve_neitherExplicitNorDefault(t *testing.T) {
	setupTempStore(t)
	_, err := ResolveAccount("", "")
	require.Error(t, err)
}

func TestStore_LoadAccount_populatesPaths(t *testing.T) {
	setupTempStore(t)
	require.NoError(t, AddAccount(Meta{Name: "alice", State: StateNEW, APIID: 1, APIHash: "h"}))
	a, err := LoadAccount("alice")
	require.NoError(t, err)
	require.Equal(t, "alice", a.Meta.Name)
	require.Equal(t, AccountDir("alice"), a.Dir)
	require.Equal(t, LockFile("alice"), a.Lock)
	require.Equal(t, SessionFile("alice"), a.Sess)
}

func TestRemoveAccount_Idempotent(t *testing.T) {
	setupTempStore(t)
	// Create a real account directory tree.
	require.NoError(t, AddAccount(Meta{Name: "alice", State: StateNEW, APIID: 1, APIHash: "h"}))
	require.DirExists(t, AccountDir("alice"))

	// First call removes the directory.
	require.NoError(t, RemoveAccount("alice"))
	require.NoDirExists(t, AccountDir("alice"))

	// Second call on a missing directory is still a no-op.
	require.NoError(t, RemoveAccount("alice"))

	// Call on a name that was never created is also fine.
	require.NoError(t, RemoveAccount("nonexistent"))
}
